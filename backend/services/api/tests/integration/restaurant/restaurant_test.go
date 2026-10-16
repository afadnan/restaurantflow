package restaurant_test

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
	restaurantdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	restaurantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/repository"
	restaurantusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type integrationDB struct {
	pool *pgxpool.Pool
	db   *postgres.DB
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

	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))

	return &integrationDB{
		pool: pool,
		db:   postgres.NewDB(pool),
	}
}

func (db *integrationDB) createTenant(t *testing.T) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tenantID := uuid.New()
	slug := "restaurant-it-" + tenantID.String()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, $2, $3)
	`, tenantID, "Restaurant Integration Tenant", slug)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cleanupCancel()

		_, _ = db.pool.Exec(
			cleanupCtx,
			`DELETE FROM tenants WHERE id = $1`,
			tenantID,
		)
	})

	return tenantID
}

func (db *integrationDB) createRestaurant(
	t *testing.T,
	tenantID uuid.UUID,
	name string,
	slug string,
	eventRepo sharedevents.EventRepository,
) *restaurantdomain.Restaurant {
	t.Helper()

	if eventRepo == nil {
		eventRepo = events.NewPostgresEventRepository()
	}

	uc := &restaurantusecase.CreateRestaurantUseCase{
		DB:          db.db,
		Restaurants: restaurantrepository.NewPostgresRestaurantRepository(),
		Events:      eventRepo,
		Clock:       restaurantusecase.SystemClock{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	restaurant, err := uc.Execute(ctx, restaurantusecase.CreateRestaurantInput{
		TenantID: tenantID,
		Name:     name,
		Slug:     slug,
		City:     "Kolkata",
		Country:  "India",
	})
	require.NoError(t, err)
	require.NotNil(t, restaurant)

	return restaurant
}

func (db *integrationDB) countOutboxEvents(
	t *testing.T,
	tenantID uuid.UUID,
	aggregateID uuid.UUID,
	eventType string,
) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.db.BeginTx(ctx, tenantID)
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

type failingEventRepository struct {
	err error
}

func (r failingEventRepository) Append(
	context.Context,
	sharedevents.Transaction,
	sharedevents.DomainEvent,
) error {
	return r.err
}

func TestIntegrationRestaurantCreateGetListUpdateAndDeactivate(t *testing.T) {
	integration := newIntegrationDB(t)
	tenantID := integration.createTenant(t)

	repository := restaurantrepository.NewPostgresRestaurantRepository()
	eventRepository := events.NewPostgresEventRepository()

	createUC := &restaurantusecase.CreateRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
		Events:      eventRepository,
		Clock:       restaurantusecase.SystemClock{},
	}
	getUC := &restaurantusecase.GetRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
	}
	listUC := &restaurantusecase.ListRestaurantsUseCase{
		DB:          integration.db,
		Restaurants: repository,
	}
	updateUC := &restaurantusecase.UpdateRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
		Events:      eventRepository,
		Clock:       restaurantusecase.SystemClock{},
	}
	deactivateUC := &restaurantusecase.DeactivateRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
		Events:      eventRepository,
		Clock:       restaurantusecase.SystemClock{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	created, err := createUC.Execute(ctx, restaurantusecase.CreateRestaurantInput{
		TenantID:    tenantID,
		Name:        "Integration Restaurant",
		Slug:        "integration-restaurant",
		Description: "Initial description",
		Phone:       "+91-1111111111",
		Email:       "first@example.test",
		City:        "Kolkata",
		Country:     "India",
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, tenantID, created.TenantID.UUID())
	require.Equal(t, "Integration Restaurant", created.Name.String())
	require.Equal(t, "integration-restaurant", created.Slug.String())
	require.Equal(t, "active", string(created.Status))

	require.Equal(t, 1, integration.countOutboxEvents(
		t,
		tenantID,
		created.ID.UUID(),
		"restaurant.created",
	))

	// Get the restaurant by ID.
	fetched, err := getUC.Execute(ctx, restaurantusecase.GetRestaurantInput{
		TenantID:     tenantID,
		RestaurantID: created.ID.UUID(),
	})
	require.NoError(t, err)
	require.Equal(t, created.ID.UUID(), fetched.ID.UUID())
	require.Equal(t, "Integration Restaurant", fetched.Name.String())

	// Add a second restaurant, then verify list pagination.
	_, err = createUC.Execute(ctx, restaurantusecase.CreateRestaurantInput{
		TenantID: tenantID,
		Name:     "Second Restaurant",
		Slug:     "second-restaurant",
		Country:  "India",
	})
	require.NoError(t, err)

	listed, err := listUC.Execute(ctx, restaurantusecase.ListRestaurantsInput{
		TenantID: tenantID,
		Limit:    1,
		Offset:   0,
	})
	require.NoError(t, err)
	require.Len(t, listed.Restaurants, 1)
	require.Equal(t, 1, listed.Limit)
	require.Equal(t, 0, listed.Offset)

	// Update the first restaurant and verify persisted data.
	updated, err := updateUC.Execute(ctx, restaurantusecase.UpdateRestaurantInput{
		TenantID:     tenantID,
		RestaurantID: created.ID.UUID(),
		Name:         "Updated Integration Restaurant",
		Slug:         "updated-integration-restaurant",
		Description:  "Updated description",
		Phone:        "+91-2222222222",
		Email:        "updated@example.test",
		City:         "Asansol",
		Country:      "India",
	})
	require.NoError(t, err)
	require.Equal(t, "Updated Integration Restaurant", updated.Name.String())
	require.Equal(t, "updated-integration-restaurant", updated.Slug.String())
	require.Equal(t, "Updated description", updated.Description)
	require.Equal(t, "Asansol", updated.City)

	require.Equal(t, 1, integration.countOutboxEvents(
		t,
		tenantID,
		created.ID.UUID(),
		"restaurant.updated",
	))

	// Deactivate and verify the final state.
	deactivated, err := deactivateUC.Execute(
		ctx,
		restaurantusecase.DeactivateRestaurantInput{
			TenantID:     tenantID,
			RestaurantID: created.ID.UUID(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, "inactive", string(deactivated.Status))

	require.Equal(t, 1, integration.countOutboxEvents(
		t,
		tenantID,
		created.ID.UUID(),
		"restaurant.deactivated",
	))
}

func TestIntegrationRestaurantTenantIsolation(t *testing.T) {
	integration := newIntegrationDB(t)
	tenantA := integration.createTenant(t)
	tenantB := integration.createTenant(t)

	restaurantA := integration.createRestaurant(
		t,
		tenantA,
		"Tenant A Restaurant",
		"shared-slug",
		nil,
	)

	// The database schema permits the same slug for a different tenant.
	restaurantB := integration.createRestaurant(
		t,
		tenantB,
		"Tenant B Restaurant",
		"shared-slug",
		nil,
	)
	require.NotEqual(t, restaurantA.ID.UUID(), restaurantB.ID.UUID())

	getUC := &restaurantusecase.GetRestaurantUseCase{
		DB:          integration.db,
		Restaurants: restaurantrepository.NewPostgresRestaurantRepository(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tenant B cannot retrieve tenant A's restaurant using the use case.
	_, err := getUC.Execute(ctx, restaurantusecase.GetRestaurantInput{
		TenantID:     tenantB,
		RestaurantID: restaurantA.ID.UUID(),
	})
	require.ErrorIs(t, err, restaurantdomain.ErrRestaurantNotFound)

	// Tenant B's list must not include tenant A's restaurant.
	listUC := &restaurantusecase.ListRestaurantsUseCase{
		DB:          integration.db,
		Restaurants: restaurantrepository.NewPostgresRestaurantRepository(),
	}
	listed, err := listUC.Execute(ctx, restaurantusecase.ListRestaurantsInput{
		TenantID: tenantB,
		Limit:    100,
		Offset:   0,
	})
	require.NoError(t, err)
	require.NotEmpty(t, listed.Restaurants)

	for _, restaurant := range listed.Restaurants {
		require.Equal(t, tenantB, restaurant.TenantID.UUID())
		require.NotEqual(t, restaurantA.ID.UUID(), restaurant.ID.UUID())
	}
}

func TestIntegrationRestaurantDuplicateSlugWithinTenant(t *testing.T) {
	integration := newIntegrationDB(t)
	tenantID := integration.createTenant(t)

	integration.createRestaurant(
		t,
		tenantID,
		"Original Restaurant",
		"duplicate-slug",
		nil,
	)

	repository := restaurantrepository.NewPostgresRestaurantRepository()
	createUC := &restaurantusecase.CreateRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
		Events:      events.NewPostgresEventRepository(),
		Clock:       restaurantusecase.SystemClock{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := createUC.Execute(ctx, restaurantusecase.CreateRestaurantInput{
		TenantID: tenantID,
		Name:     "Duplicate Restaurant",
		Slug:     "duplicate-slug",
		Country:  "India",
	})
	require.Error(t, err)
}

func TestIntegrationRestaurantCreateRollsBackWhenEventAppendFails(t *testing.T) {
	integration := newIntegrationDB(t)
	tenantID := integration.createTenant(t)

	appendErr := errors.New("simulated outbox failure")
	repository := restaurantrepository.NewPostgresRestaurantRepository()
	createUC := &restaurantusecase.CreateRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
		Events:      failingEventRepository{err: appendErr},
		Clock:       restaurantusecase.SystemClock{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := createUC.Execute(ctx, restaurantusecase.CreateRestaurantInput{
		TenantID: tenantID,
		Name:     "Must Roll Back",
		Slug:     "must-roll-back",
		Country:  "India",
	})
	require.ErrorIs(t, err, appendErr)

	// Confirm that the restaurant insert was rolled back.
	tx, err := integration.db.BeginTx(ctx, tenantID)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	var count int
	err = tx.Raw().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM restaurants
		WHERE tenant_id = $1
		  AND slug = $2
	`, tenantID, "must-roll-back").Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestIntegrationRestaurantUpdateAndDeactivateRollBackWhenEventAppendFails(t *testing.T) {
	integration := newIntegrationDB(t)
	tenantID := integration.createTenant(t)

	restaurant := integration.createRestaurant(
		t,
		tenantID,
		"Original Name",
		"original-name",
		nil,
	)

	repository := restaurantrepository.NewPostgresRestaurantRepository()
	appendErr := errors.New("simulated outbox failure")
	failingEvents := failingEventRepository{err: appendErr}

	updateUC := &restaurantusecase.UpdateRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
		Events:      failingEvents,
		Clock:       restaurantusecase.SystemClock{},
	}
	deactivateUC := &restaurantusecase.DeactivateRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
		Events:      failingEvents,
		Clock:       restaurantusecase.SystemClock{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := updateUC.Execute(ctx, restaurantusecase.UpdateRestaurantInput{
		TenantID:     tenantID,
		RestaurantID: restaurant.ID.UUID(),
		Name:         "Should Not Persist",
		Slug:         "should-not-persist",
		Country:      "India",
	})
	require.ErrorIs(t, err, appendErr)

	_, err = deactivateUC.Execute(
		ctx,
		restaurantusecase.DeactivateRestaurantInput{
			TenantID:     tenantID,
			RestaurantID: restaurant.ID.UUID(),
		},
	)
	require.ErrorIs(t, err, appendErr)

	getUC := &restaurantusecase.GetRestaurantUseCase{
		DB:          integration.db,
		Restaurants: repository,
	}
	persisted, err := getUC.Execute(ctx, restaurantusecase.GetRestaurantInput{
		TenantID:     tenantID,
		RestaurantID: restaurant.ID.UUID(),
	})
	require.NoError(t, err)
	require.Equal(t, "Original Name", persisted.Name.String())
	require.Equal(t, "original-name", persisted.Slug.String())
	require.Equal(t, "active", string(persisted.Status))
}

func TestIntegrationRestaurantRLSWithoutTenantContext(t *testing.T) {
	integration := newIntegrationDB(t)
	tenantID := integration.createTenant(t)

	integration.createRestaurant(
		t,
		tenantID,
		"RLS Restaurant",
		"rls-restaurant",
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This check assumes the configured test role is subject to RLS.
	var count int
	err := integration.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM restaurants
	`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count,
		"restaurants should be invisible without app.tenant_id; "+
			"check whether the configured database role bypasses RLS")
}

func TestIntegrationRestaurantDatabaseConnection(t *testing.T) {
	integration := newIntegrationDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, integration.pool.Ping(ctx))

	var result int
	require.NoError(t, integration.pool.QueryRow(ctx, `SELECT 1`).Scan(&result))
	require.Equal(t, 1, result)

	_ = fmt.Sprintf("%s", integration.pool.Config().ConnConfig.Database)
}
