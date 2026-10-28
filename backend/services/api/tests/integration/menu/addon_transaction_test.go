package menu_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

var errAddonEventFailure = errors.New("injected addon event failure")

type failingAddonEvents struct{}

func (failingAddonEvents) Append(
	_ context.Context,
	_ sharedevents.Transaction,
	_ sharedevents.DomainEvent,
) error {
	return errAddonEventFailure
}

func TestIntegrationMenuAddonTransaction(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	addonRepo := repository.NewPostgresAddonRepository()

	dependencies := func(
		eventRepo sharedevents.EventRepository,
	) usecase.AddonUseCaseDependencies {
		return usecase.AddonUseCaseDependencies{
			DB:     integration.db,
			Addons: addonRepo,
			Events: eventRepo,
		}
	}

	realDeps := dependencies(events.NewPostgresEventRepository())
	failingDeps := dependencies(failingAddonEvents{})

	createInput := func(name string) usecase.CreateAddonInput {
		return usecase.CreateAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			Name:         name,
			Description:  "Transaction test addon",
			Price:        "25.00",
		}
	}

	getAddon := func(id uuid.UUID) *domain.Addon {
		t.Helper()

		uc := &usecase.GetAddonUseCase{
			AddonUseCaseDependencies: realDeps,
		}

		addon, err := uc.Execute(ctx, usecase.GetAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			AddonID:      id,
		})
		require.NoError(t, err)
		return addon
	}

	countByName := func(name string) int {
		t.Helper()

		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		var count int
		err = tx.Raw().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM menu_addons
			WHERE tenant_id = $1
			  AND restaurant_id = $2
			  AND name = $3
		`, fixture.tenantID, fixture.restaurantID, name).Scan(&count)
		require.NoError(t, err)
		return count
	}

	var addonID uuid.UUID

	t.Run("create commits addon and event", func(t *testing.T) {
		uc := &usecase.CreateAddonUseCase{
			AddonUseCaseDependencies: realDeps,
		}

		addon, err := uc.Execute(
			ctx, createInput("Transaction Addon"),
		)
		require.NoError(t, err)
		require.NotNil(t, addon)

		addonID = addon.ID
		require.Equal(t, 1, countByName("Transaction Addon"))
		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonCreated),
		))
	})

	t.Run("create rolls back when event fails", func(t *testing.T) {
		uc := &usecase.CreateAddonUseCase{
			AddonUseCaseDependencies: failingDeps,
		}

		addon, err := uc.Execute(
			ctx, createInput("Failed Transaction Addon"),
		)
		require.ErrorIs(t, err, errAddonEventFailure)
		require.Nil(t, addon)
		require.Zero(t, countByName("Failed Transaction Addon"))
	})

	t.Run("update commits addon and event", func(t *testing.T) {
		uc := &usecase.UpdateAddonUseCase{
			AddonUseCaseDependencies: realDeps,
		}

		updated, err := uc.Execute(ctx, usecase.UpdateAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			AddonID:      addonID,
			Name:         "Updated Transaction Addon",
			Description:  "Updated",
			Price:        "35.00",
		})
		require.NoError(t, err)
		require.Equal(t, "Updated Transaction Addon", updated.Name)
		require.Equal(t, "35.00", getAddon(addonID).Price)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonUpdated),
		))
	})

	t.Run("update rolls back when event fails", func(t *testing.T) {
		before := getAddon(addonID)

		uc := &usecase.UpdateAddonUseCase{
			AddonUseCaseDependencies: failingDeps,
		}

		updated, err := uc.Execute(ctx, usecase.UpdateAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			AddonID:      addonID,
			Name:         "Should Not Persist",
			Description:  "Failed update",
			Price:        "99.00",
		})
		require.ErrorIs(t, err, errAddonEventFailure)
		require.Nil(t, updated)
		require.Equal(t, before, getAddon(addonID))

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonUpdated),
		))
	})

	t.Run("deactivate rolls back when event fails", func(t *testing.T) {
		before := getAddon(addonID)

		uc := &usecase.DeactivateAddonUseCase{
			AddonUseCaseDependencies: failingDeps,
		}

		deactivated, err := uc.Execute(
			ctx, usecase.DeactivateAddonInput{
				TenantID:     fixture.tenantID,
				RestaurantID: fixture.restaurantID,
				AddonID:      addonID,
			},
		)
		require.ErrorIs(t, err, errAddonEventFailure)
		require.Nil(t, deactivated)
		require.Equal(t, before, getAddon(addonID))
		require.True(t, getAddon(addonID).IsActive)

		require.Zero(t, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonDeactivated),
		))
	})

	t.Run("deactivate commits addon and event", func(t *testing.T) {
		uc := &usecase.DeactivateAddonUseCase{
			AddonUseCaseDependencies: realDeps,
		}

		deactivated, err := uc.Execute(
			ctx, usecase.DeactivateAddonInput{
				TenantID:     fixture.tenantID,
				RestaurantID: fixture.restaurantID,
				AddonID:      addonID,
			},
		)
		require.NoError(t, err)
		require.False(t, deactivated.IsActive)
		require.False(t, getAddon(addonID).IsActive)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, addonID,
			string(domain.EventTypeAddonDeactivated),
		))
	})

	t.Run("inactive addon cannot be updated", func(t *testing.T) {
		uc := &usecase.UpdateAddonUseCase{
			AddonUseCaseDependencies: realDeps,
		}

		_, err := uc.Execute(ctx, usecase.UpdateAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			AddonID:      addonID,
			Name:         "Inactive Update",
			Price:        "50.00",
		})
		require.ErrorIs(t, err, domain.ErrAddonInactive)
	})
}
