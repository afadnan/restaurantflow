package order_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	orderdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	orderrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/order/repository"
	orderusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/order/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type integrationDB struct {
	pool *pgxpool.Pool
}

func newIntegrationDB(t *testing.T) *integrationDB {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load()
	require.NoError(t, err)

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	require.NoError(t, pool.Ping(ctx))

	return &integrationDB{
		pool: pool,
	}
}

type orderFixture struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	CategoryID   uuid.UUID
	MenuItemID   uuid.UUID
	IngredientID uuid.UUID
	InventoryQty string
	RequiredQty  string
}

func createFixture(
	t *testing.T,
	db *integrationDB,
	initialInventory string,
	requiredQuantity string,
) orderFixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tenantID := uuid.New()
	restaurantID := uuid.New()
	categoryID := uuid.New()
	menuItemID := uuid.New()
	ingredientID := uuid.New()

	_, err := db.pool.Exec(
		ctx,
		`
		INSERT INTO tenants (
			id,
			name,
			slug
		)
		VALUES ($1, $2, $3)
		`,
		tenantID,
		"Integration Test Tenant",
		"integration-"+tenantID.String(),
	)
	require.NoError(t, err)

	tx, err := db.pool.Begin(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tx.Rollback(ctx)
	})

	_, err = tx.Exec(
		ctx,
		`SELECT set_config('app.tenant_id', $1, true)`,
		tenantID.String(),
	)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO restaurants (
			id,
			tenant_id,
			name,
			slug
		)
		VALUES ($1, $2, $3, $4)
		`,
		restaurantID,
		tenantID,
		"Integration Restaurant",
		"restaurant-"+restaurantID.String(),
	)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO menu_categories (
			id,
			tenant_id,
			restaurant_id,
			name
		)
		VALUES ($1, $2, $3, $4)
		`,
		categoryID,
		tenantID,
		restaurantID,
		"Integration Category",
	)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO menu_items (
			id,
			tenant_id,
			restaurant_id,
			category_id,
			name,
			sku,
			price
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		menuItemID,
		tenantID,
		restaurantID,
		categoryID,
		"Integration Burger",
		"TEST-"+menuItemID.String(),
		100.00,
	)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO ingredients (
			id,
			tenant_id,
			restaurant_id,
			name,
			sku,
			unit
		)
		VALUES ($1, $2, $3, $4, $5, 'gram')
		`,
		ingredientID,
		tenantID,
		restaurantID,
		"Integration Flour",
		"ING-"+ingredientID.String(),
	)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO menu_item_ingredients (
			tenant_id,
			menu_item_id,
			ingredient_id,
			quantity_required
		)
		VALUES ($1, $2, $3, $4)
		`,
		tenantID,
		menuItemID,
		ingredientID,
		requiredQuantity,
	)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO inventory (
			id,
			tenant_id,
			restaurant_id,
			ingredient_id,
			quantity,
			low_stock_threshold
		)
		VALUES ($1, $2, $3, $4, $5, 0)
		`,
		uuid.New(),
		tenantID,
		restaurantID,
		ingredientID,
		initialInventory,
	)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	return orderFixture{
		TenantID:     tenantID,
		RestaurantID: restaurantID,
		CategoryID:   categoryID,
		MenuItemID:   menuItemID,
		IngredientID: ingredientID,
		InventoryQty: initialInventory,
		RequiredQty:  requiredQuantity,
	}
}

func newCreateOrderUseCase(
	db *integrationDB,
) *orderusecase.CreateOrderUseCase {
	return &orderusecase.CreateOrderUseCase{
		DB:        postgres.NewDB(db.pool),
		Orders:    orderrepository.NewPostgresOrderRepository(),
		KDS:       orderrepository.NewPostgresKDSRepository(),
		Inventory: orderrepository.NewPostgresInventoryRepository(),
		Events:    events.NewPostgresEventRepository(),
		Clock: func() time.Time {
			return time.Date(
				2026,
				time.January,
				1,
				12,
				0,
				0,
				0,
				time.UTC,
			)
		},
	}
}

type failingEventRepository struct {
	err error
}

func (r failingEventRepository) Append(
	ctx context.Context,
	tx sharedevents.Transaction,
	event sharedevents.DomainEvent,
) error {
	return r.err
}

func TestCreateOrder_RollsBackOnEventFailure(t *testing.T) {
	db := newIntegrationDB(t)

	fixture := createFixture(
		t,
		db,
		"2.000",
		"0.500",
	)

	expectedErr := fmt.Errorf("forced outbox append failure")

	uc := newCreateOrderUseCase(db)

	// Replace the real event repository with one that fails
	// after Orders.Create, KDS.Create, and inventory deduction
	// have already succeeded inside the transaction.
	uc.Events = failingEventRepository{
		err: expectedErr,
	}

	_, err := uc.Execute(
		context.Background(),
		createOrderInput(fixture, 1),
	)

	require.ErrorIs(t, err, expectedErr)

	// The order was inserted before the forced event failure.
	// It must have been rolled back.
	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "orders"),
	)

	// The order item was inserted in the same transaction.
	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "order_items"),
	)

	// The KDS ticket was inserted in the same transaction.
	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "kds_tickets"),
	)

	// The outbox event must not exist because Append failed.
	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "outbox_events"),
	)

	// Inventory deduction happened before the forced event failure,
	// so it must also have been rolled back.
	require.Equal(
		t,
		"2.000",
		readInventoryQuantity(t, db, fixture),
	)
}

func createOrderInput(
	f orderFixture,
	quantity int32,
) orderusecase.CreateOrderInput {
	return orderusecase.CreateOrderInput{
		TenantID:      f.TenantID,
		RestaurantID:  f.RestaurantID,
		OrderType:     orderdomain.OrderTypeDineIn,
		PaymentStatus: orderdomain.PaymentStatusPending,
		Currency:      "INR",
		Items: []orderusecase.CreateOrderItemInput{
			{
				ProductID: f.MenuItemID,
				Name:      "Integration Burger",
				Quantity:  quantity,
				UnitPrice: 10000,
				Notes:     "",
			},
		},
	}
}

func countForTenant(
	t *testing.T,
	db *integrationDB,
	tenantID uuid.UUID,
	table string,
) int {
	t.Helper()

	require.Contains(
		t,
		[]string{
			"orders",
			"order_items",
			"kds_tickets",
			"outbox_events",
		},
		table,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		`SELECT set_config('app.tenant_id', $1, true)`,
		tenantID.String(),
	)
	require.NoError(t, err)

	var count int

	err = tx.QueryRow(
		ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table),
	).Scan(&count)
	require.NoError(t, err)

	return count
}

type persistedOrderIDs struct {
	OrderID          uuid.UUID
	OrderItemOrderID uuid.UUID
	KDSTicketOrderID uuid.UUID
	OutboxEventID    uuid.UUID
	AggregateID      uuid.UUID
}

func readPersistedOrderIDs(
	t *testing.T,
	db *integrationDB,
	tenantID uuid.UUID,
) persistedOrderIDs {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		`SELECT set_config('app.tenant_id', $1, true)`,
		tenantID.String(),
	)
	require.NoError(t, err)

	var result persistedOrderIDs

	// The order aggregate's identity.
	err = tx.QueryRow(
		ctx,
		`
		SELECT id
		FROM orders
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
		`,
		tenantID,
	).Scan(&result.OrderID)
	require.NoError(t, err)

	// order_items.id is a separate row identity.
	// The relationship to the order is order_items.order_id.
	err = tx.QueryRow(
		ctx,
		`
		SELECT order_id
		FROM order_items
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
		`,
		tenantID,
	).Scan(&result.OrderItemOrderID)
	require.NoError(t, err)

	// kds_tickets.id is a separate ticket identity.
	// The relationship to the order is kds_tickets.order_id.
	err = tx.QueryRow(
		ctx,
		`
		SELECT order_id
		FROM kds_tickets
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
		`,
		tenantID,
	).Scan(&result.KDSTicketOrderID)
	require.NoError(t, err)

	// outbox_events.id is the event-row identity.
	// aggregate_id identifies the order aggregate.
	err = tx.QueryRow(
		ctx,
		`
		SELECT id, aggregate_id
		FROM outbox_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
		`,
		tenantID,
	).Scan(
		&result.OutboxEventID,
		&result.AggregateID,
	)
	require.NoError(t, err)

	return result
}

func readInventoryQuantity(
	t *testing.T,
	db *integrationDB,
	f orderFixture,
) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		`SELECT set_config('app.tenant_id', $1, true)`,
		f.TenantID.String(),
	)
	require.NoError(t, err)

	var quantity string

	err = tx.QueryRow(
		ctx,
		`
		SELECT quantity::TEXT
		FROM inventory
		WHERE tenant_id = $1
		  AND restaurant_id = $2
		  AND ingredient_id = $3
		`,
		f.TenantID,
		f.RestaurantID,
		f.IngredientID,
	).Scan(&quantity)
	require.NoError(t, err)

	return quantity
}

func TestCreateOrder_AtomicSuccess(t *testing.T) {
	db := newIntegrationDB(t)

	fixture := createFixture(
		t,
		db,
		"2.000",
		"0.500",
	)

	uc := newCreateOrderUseCase(db)

	order, err := uc.Execute(
		context.Background(),
		createOrderInput(fixture, 2),
	)

	require.NoError(t, err)
	require.NotNil(t, order)
	require.NotEqual(t, uuid.Nil, order.ID.UUID())

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixture.TenantID, "orders"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixture.TenantID, "order_items"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixture.TenantID, "kds_tickets"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixture.TenantID, "outbox_events"),
	)

	ids := readPersistedOrderIDs(
		t,
		db,
		fixture.TenantID,
	)

	// All related records must point to the same order aggregate.
	require.Equal(
		t,
		ids.OrderID,
		ids.OrderItemOrderID,
	)

	require.Equal(
		t,
		ids.OrderID,
		ids.KDSTicketOrderID,
	)

	require.Equal(
		t,
		ids.OrderID,
		ids.AggregateID,
	)

	// The returned domain order must have the same ID
	// that was persisted in orders.id.
	require.Equal(
		t,
		order.ID.UUID(),
		ids.OrderID,
	)

	// 2 × 0.500 = 1.000 deducted from 2.000.
	require.Equal(
		t,
		"1.000",
		readInventoryQuantity(t, db, fixture),
	)
}

func TestCreateOrder_RollsBackOnInventoryFailure(t *testing.T) {
	db := newIntegrationDB(t)

	fixture := createFixture(
		t,
		db,
		"1.000",
		"0.500",
	)

	uc := newCreateOrderUseCase(db)

	_, err := uc.Execute(
		context.Background(),
		createOrderInput(fixture, 3),
	)

	require.Error(t, err)

	// 3 × 0.500 = 1.500 required,
	// but only 1.000 exists.
	require.Contains(
		t,
		err.Error(),
		"insufficient inventory",
	)

	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "orders"),
	)

	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "order_items"),
	)

	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "kds_tickets"),
	)

	require.Equal(
		t,
		0,
		countForTenant(t, db, fixture.TenantID, "outbox_events"),
	)

	// The failed transaction must not partially deduct inventory.
	require.Equal(
		t,
		"1.000",
		readInventoryQuantity(t, db, fixture),
	)
}

func TestCreateOrder_TenantIsolation(t *testing.T) {
	db := newIntegrationDB(t)

	fixtureA := createFixture(
		t,
		db,
		"2.000",
		"0.500",
	)

	fixtureB := createFixture(
		t,
		db,
		"2.000",
		"0.500",
	)

	uc := newCreateOrderUseCase(db)

	orderA, err := uc.Execute(
		context.Background(),
		createOrderInput(fixtureA, 1),
	)
	require.NoError(t, err)

	orderB, err := uc.Execute(
		context.Background(),
		createOrderInput(fixtureB, 1),
	)
	require.NoError(t, err)

	require.NotEqual(
		t,
		orderA.ID.UUID(),
		orderB.ID.UUID(),
	)

	// Tenant A sees only its own order-related records.
	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureA.TenantID, "orders"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureA.TenantID, "order_items"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureA.TenantID, "kds_tickets"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureA.TenantID, "outbox_events"),
	)

	// Tenant B sees only its own order-related records.
	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureB.TenantID, "orders"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureB.TenantID, "order_items"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureB.TenantID, "kds_tickets"),
	)

	require.Equal(
		t,
		1,
		countForTenant(t, db, fixtureB.TenantID, "outbox_events"),
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	// Verify tenant A cannot see tenant B's order.
	tx, err := db.pool.Begin(ctx)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`SELECT set_config('app.tenant_id', $1, true)`,
		fixtureA.TenantID.String(),
	)
	require.NoError(t, err)

	var visibleToA int

	err = tx.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM orders
		WHERE id IN ($1, $2)
		`,
		orderA.ID.UUID(),
		orderB.ID.UUID(),
	).Scan(&visibleToA)
	require.NoError(t, err)

	require.Equal(t, 1, visibleToA)

	require.NoError(t, tx.Rollback(ctx))

	// Verify tenant B cannot see tenant A's order.
	tx, err = db.pool.Begin(ctx)
	require.NoError(t, err)

	_, err = tx.Exec(
		ctx,
		`SELECT set_config('app.tenant_id', $1, true)`,
		fixtureB.TenantID.String(),
	)
	require.NoError(t, err)

	var visibleToB int

	err = tx.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM orders
		WHERE id IN ($1, $2)
		`,
		orderA.ID.UUID(),
		orderB.ID.UUID(),
	).Scan(&visibleToB)
	require.NoError(t, err)

	require.Equal(t, 1, visibleToB)

	require.NoError(t, tx.Rollback(ctx))
}

func TestIntegrationDBRequiresTenantContextForRLS(t *testing.T) {
	db := newIntegrationDB(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	var count int

	err := db.pool.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM orders`,
	).Scan(&count)

	require.NoError(t, err)

	// With no app.tenant_id set, tenant-scoped RLS
	// must expose zero rows.
	require.Equal(t, 0, count)
}
