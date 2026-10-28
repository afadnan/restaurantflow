package menu_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

// Overrides only the tenant identity used by the event repository.
// All persistence still runs through the real PostgreSQL event repository.
type invalidTenantEvent struct {
	sharedevents.DomainEvent
	tenantID uuid.UUID
}

func (e invalidTenantEvent) TenantID() uuid.UUID {
	return e.tenantID
}

type postgresFailingAddonEvents struct {
	real            sharedevents.EventRepository
	invalidTenantID uuid.UUID
}

func (r postgresFailingAddonEvents) Append(
	ctx context.Context,
	tx sharedevents.Transaction,
	event sharedevents.DomainEvent,
) error {
	return r.real.Append(ctx, tx, invalidTenantEvent{
		DomainEvent: event,
		tenantID:    r.invalidTenantID,
	})
}

func TestIntegrationMenuAddonRealOutboxInsertFailure(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(
		context.Background(), 30*time.Second,
	)
	defer cancel()

	invalidTenantID := uuid.New()

	// Ensure the deliberately invalid tenant does not exist.
	var tenantExists bool
	err := integration.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM tenants WHERE id = $1
		)
	`, invalidTenantID).Scan(&tenantExists)
	require.NoError(t, err)
	require.False(t, tenantExists)

	uc := &usecase.CreateAddonUseCase{
		AddonUseCaseDependencies: usecase.AddonUseCaseDependencies{
			DB:     integration.db,
			Addons: repository.NewPostgresAddonRepository(),
			Events: postgresFailingAddonEvents{
				real:            events.NewPostgresEventRepository(),
				invalidTenantID: invalidTenantID,
			},
		},
	}

	addonName := "Real Outbox Failure " + uuid.NewString()

	created, err := uc.Execute(ctx, usecase.CreateAddonInput{
		TenantID:     fixture.tenantID,
		RestaurantID: fixture.restaurantID,
		Name:         addonName,
		Description:  "Must roll back on actual SQL failure",
		Price:        "25.00",
	})

	require.Nil(t, created)
	require.Error(t, err)

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)

	t.Logf(
		"PostgreSQL error: code=%s message=%s detail=%s constraint=%s table=%s",
		pgErr.Code,
		pgErr.Message,
		pgErr.Detail,
		pgErr.ConstraintName,
		pgErr.TableName,
	)

	require.Equal(t, "42501", pgErr.Code)
	require.Contains(
		t,
		pgErr.Message,
		`violates row-level security policy for table "outbox_events"`,
	)

	// Verify that the earlier Add-on INSERT was rolled back.
	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	var addonCount int
	err = tx.Raw().QueryRow(
		ctx, `
		SELECT COUNT(*)
		FROM menu_addons
		WHERE tenant_id = $1
		  AND restaurant_id = $2
		  AND name = $3
	`,
		fixture.tenantID,
		fixture.restaurantID,
		addonName,
	).Scan(&addonCount)
	require.NoError(t, err)
	require.Zero(t, addonCount)

	// Confirm the attempted event was not committed.
	var eventCount int
	err = tx.Raw().QueryRow(
		ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE tenant_id = $1
		  AND event_type = $2
	`,
		invalidTenantID,
		string(domain.EventTypeAddonCreated),
	).Scan(&eventCount)
	require.NoError(t, err)
	require.Zero(t, eventCount)
}
