package menu_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	menudomain "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	menurepository "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	menuusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type variantIntegration struct {
	pool *pgxpool.Pool
	db   *postgres.DB
}

type variantFixture struct {
	tenantID     uuid.UUID
	restaurantID uuid.UUID
	itemID       uuid.UUID
}

func newVariantIntegration(t *testing.T) *variantIntegration {
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

	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	return &variantIntegration{
		pool: pool,
		db:   postgres.NewDB(pool),
	}
}

func (integration *variantIntegration) createFixture(t *testing.T) variantFixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fixture := variantFixture{
		tenantID:     uuid.New(),
		restaurantID: uuid.New(),
		itemID:       uuid.New(),
	}
	categoryID := uuid.New()

	_, err := integration.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, $2, $3)
	`, fixture.tenantID, "Variant Integration Tenant",
		"variant-it-"+fixture.tenantID.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cleanupCancel()

		_, _ = integration.pool.Exec(
			cleanupCtx,
			`DELETE FROM tenants WHERE id = $1`,
			fixture.tenantID,
		)
	})

	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO restaurants (id, tenant_id, name, slug)
		VALUES ($1, $2, $3, $4)
	`, fixture.restaurantID, fixture.tenantID,
		"Variant Integration Restaurant",
		"variant-restaurant-"+fixture.restaurantID.String())
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO menu_categories (
			id, tenant_id, restaurant_id, name
		)
		VALUES ($1, $2, $3, $4)
	`, categoryID, fixture.tenantID, fixture.restaurantID,
		"Variant Integration Category")
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO menu_items (
			id, tenant_id, restaurant_id, category_id,
			name, sku, price
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, fixture.itemID, fixture.tenantID, fixture.restaurantID,
		categoryID, "Variant Integration Item",
		"VARIANT-"+fixture.itemID.String(), 100.00)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))
	return fixture
}

type failingVariantEvents struct {
	err error
}

func (repo failingVariantEvents) Append(
	context.Context,
	sharedevents.Transaction,
	sharedevents.DomainEvent,
) error {
	return repo.err
}

func (integration *variantIntegration) countEvents(
	t *testing.T,
	tenantID, aggregateID uuid.UUID,
	eventType string,
) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := integration.db.BeginTx(ctx, tenantID)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	var count int
	err = tx.Raw().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE tenant_id = $1
		  AND aggregate_id = $2
		  AND event_type = $3
	`, tenantID, aggregateID, eventType).Scan(&count)
	require.NoError(t, err)

	return count
}

func TestIntegrationMenuItemVariantCRUDAndTenantBoundary(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	repo := menurepository.NewPostgresVariantRepository()
	eventRepo := events.NewPostgresEventRepository()

	createUC := &menuusecase.CreateVariantUseCase{
		VariantUseCaseDependencies: menuusecase.VariantUseCaseDependencies{
			DB: integration.db, Variants: repo, Events: eventRepo,
		},
	}
	getUC := &menuusecase.GetVariantUseCase{
		VariantUseCaseDependencies: menuusecase.VariantUseCaseDependencies{
			DB: integration.db, Variants: repo,
		},
	}
	listUC := &menuusecase.ListVariantsUseCase{
		VariantUseCaseDependencies: menuusecase.VariantUseCaseDependencies{
			DB: integration.db, Variants: repo,
		},
	}
	updateUC := &menuusecase.UpdateVariantUseCase{
		VariantUseCaseDependencies: menuusecase.VariantUseCaseDependencies{
			DB: integration.db, Variants: repo, Events: eventRepo,
		},
	}
	deactivateUC := &menuusecase.DeactivateVariantUseCase{
		VariantUseCaseDependencies: menuusecase.VariantUseCaseDependencies{
			DB: integration.db, Variants: repo, Events: eventRepo,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	created, err := createUC.Execute(ctx, menuusecase.CreateVariantInput{
		TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
		ItemID: fixture.itemID, Name: "Large", PriceDelta: "15.00",
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "Large", created.Name)
	require.True(t, created.IsActive)
	require.Equal(t, fixture.itemID, created.ItemID)
	require.Equal(t, 1, integration.countEvents(
		t, fixture.tenantID, created.ID, string(menudomain.EventTypeVariantCreated),
	))

	fetched, err := getUC.Execute(ctx, menuusecase.GetVariantInput{
		TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
		ItemID: fixture.itemID, VariantID: created.ID,
	})
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
	require.Equal(t, "Large", fetched.Name)

	listed, err := listUC.Execute(ctx, menuusecase.ListVariantsInput{
		TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
		ItemID: fixture.itemID,
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.ID, listed[0].ID)

	updated, err := updateUC.Execute(ctx, menuusecase.UpdateVariantInput{
		TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
		ItemID: fixture.itemID, VariantID: created.ID,
		Name: "Family Size", PriceDelta: "25.00",
	})
	require.NoError(t, err)
	require.Equal(t, "Family Size", updated.Name)
	require.Equal(t, "25.00", updated.PriceDelta)
	require.Equal(t, 1, integration.countEvents(
		t, fixture.tenantID, created.ID, string(menudomain.EventTypeVariantUpdated),
	))

	deactivated, err := deactivateUC.Execute(
		ctx, menuusecase.DeactivateVariantInput{
			TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
			ItemID: fixture.itemID, VariantID: created.ID,
		},
	)
	require.NoError(t, err)
	require.False(t, deactivated.IsActive)
	require.Equal(t, 1, integration.countEvents(
		t, fixture.tenantID, created.ID,
		string(menudomain.EventTypeVariantDeactivated),
	))

	// The same variant must not be readable in another tenant's context.
	otherTenantID := uuid.New()
	_, err = integration.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, $2, $3)
	`, otherTenantID, "Other Variant Tenant",
		"other-variant-"+otherTenantID.String())
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cleanupCancel()
		_, _ = integration.pool.Exec(
			cleanupCtx, `DELETE FROM tenants WHERE id = $1`, otherTenantID,
		)
	})

	_, err = getUC.Execute(ctx, menuusecase.GetVariantInput{
		TenantID: otherTenantID, RestaurantID: fixture.restaurantID,
		ItemID: fixture.itemID, VariantID: created.ID,
	})
	require.ErrorIs(t, err, menudomain.ErrVariantNotFound)
}

func TestIntegrationMenuItemVariantCreateRollsBackWhenOutboxFails(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	expectedErr := errors.New("simulated outbox failure")
	createUC := &menuusecase.CreateVariantUseCase{
		VariantUseCaseDependencies: menuusecase.VariantUseCaseDependencies{
			DB:       integration.db,
			Variants: menurepository.NewPostgresVariantRepository(),
			Events:   failingVariantEvents{err: expectedErr},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	created, err := createUC.Execute(ctx, menuusecase.CreateVariantInput{
		TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
		ItemID: fixture.itemID, Name: "Should Roll Back", PriceDelta: "5.00",
	})
	require.Nil(t, created)
	require.ErrorIs(t, err, expectedErr)

	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	var count int
	err = tx.Raw().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM menu_item_variants
		WHERE tenant_id = $1
		  AND restaurant_id = $2
		  AND menu_item_id = $3
		  AND name = $4
	`, fixture.tenantID, fixture.restaurantID, fixture.itemID,
		"Should Roll Back").Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count, fmt.Sprintf("variant persisted after outbox failure"))
}
