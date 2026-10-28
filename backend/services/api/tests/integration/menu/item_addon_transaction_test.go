package menu_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
)

func TestIntegrationMenuItemAddonTransaction(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(
		context.Background(), 60*time.Second,
	)
	defer cancel()

	addonRepo := repository.NewPostgresAddonRepository()
	linkRepo := repository.NewPostgresItemAddonRepository()

	realDeps := usecase.ItemAddonUseCaseDependencies{
		DB:         integration.db,
		ItemAddons: linkRepo,
		Events:     events.NewPostgresEventRepository(),
	}
	failingDeps := usecase.ItemAddonUseCaseDependencies{
		DB:         integration.db,
		ItemAddons: linkRepo,
		Events:     failingAddonEvents{},
	}

	createAddon := func(name string) uuid.UUID {
		t.Helper()

		addon, err := domain.NewAddon(
			fixture.tenantID,
			fixture.restaurantID,
			name, "", "15.00", time.Now().UTC(),
		)
		require.NoError(t, err)

		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		require.NoError(t, addonRepo.Create(ctx, tx, addon))
		require.NoError(t, tx.Commit(ctx))
		return addon.ID
	}

	getLink := func(addonID uuid.UUID) (*domain.ItemAddon, error) {
		uc := &usecase.GetItemAddonUseCase{
			ItemAddonUseCaseDependencies: realDeps,
		}
		return uc.Execute(ctx, usecase.GetItemAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
			AddonID:      addonID,
		})
	}

	attachInput := func(addonID uuid.UUID) usecase.AttachItemAddonInput {
		return usecase.AttachItemAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
			AddonID:      addonID,
			IsRequired:   false,
			MaxQuantity:  2,
		}
	}

	updateInput := func(addonID uuid.UUID) usecase.UpdateItemAddonInput {
		return usecase.UpdateItemAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
			AddonID:      addonID,
			IsRequired:   true,
			MaxQuantity:  3,
		}
	}

	detachInput := func(addonID uuid.UUID) usecase.DetachItemAddonInput {
		return usecase.DetachItemAddonInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
			AddonID:      addonID,
		}
	}

	// Two associations share an Item aggregate but have different
	// Add-on identities.
	addonID := createAddon("Transaction Sauce")
	rollbackAddonID := createAddon("Rollback Sauce")

	t.Run("attach commits association and event", func(t *testing.T) {
		uc := &usecase.AttachItemAddonUseCase{
			ItemAddonUseCaseDependencies: realDeps,
		}

		link, err := uc.Execute(ctx, attachInput(addonID))
		require.NoError(t, err)
		require.Equal(t, fixture.itemID, link.ItemID)
		require.Equal(t, addonID, link.AddonID)

		stored, err := getLink(addonID)
		require.NoError(t, err)
		require.Equal(t, link, stored)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemAddonAttached),
		))
	})

	t.Run("attach rolls back when event fails", func(t *testing.T) {
		uc := &usecase.AttachItemAddonUseCase{
			ItemAddonUseCaseDependencies: failingDeps,
		}

		link, err := uc.Execute(ctx, attachInput(rollbackAddonID))
		require.ErrorIs(t, err, errAddonEventFailure)
		require.Nil(t, link)

		_, err = getLink(rollbackAddonID)
		require.ErrorIs(t, err, domain.ErrItemAddonNotFound)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemAddonAttached),
		))
	})

	t.Run("update commits association and event", func(t *testing.T) {
		uc := &usecase.UpdateItemAddonUseCase{
			ItemAddonUseCaseDependencies: realDeps,
		}

		link, err := uc.Execute(ctx, updateInput(addonID))
		require.NoError(t, err)
		require.True(t, link.IsRequired)
		require.Equal(t, int32(3), link.MaxQuantity)

		stored, err := getLink(addonID)
		require.NoError(t, err)
		require.Equal(t, link, stored)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemAddonUpdated),
		))
	})

	t.Run("update rolls back when event fails", func(t *testing.T) {
		before, err := getLink(addonID)
		require.NoError(t, err)

		input := updateInput(addonID)
		input.IsRequired = false
		input.MaxQuantity = 5

		uc := &usecase.UpdateItemAddonUseCase{
			ItemAddonUseCaseDependencies: failingDeps,
		}

		link, err := uc.Execute(ctx, input)
		require.ErrorIs(t, err, errAddonEventFailure)
		require.Nil(t, link)

		after, err := getLink(addonID)
		require.NoError(t, err)
		require.Equal(t, before, after)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemAddonUpdated),
		))
	})

	t.Run("detach rolls back when event fails", func(t *testing.T) {
		uc := &usecase.DetachItemAddonUseCase{
			ItemAddonUseCaseDependencies: failingDeps,
		}

		link, err := uc.Execute(ctx, detachInput(addonID))
		require.ErrorIs(t, err, errAddonEventFailure)
		require.Nil(t, link)

		stored, err := getLink(addonID)
		require.NoError(t, err)
		require.NotNil(t, stored)

		require.Zero(t, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemAddonDetached),
		))
	})

	t.Run("detach commits deletion and event", func(t *testing.T) {
		uc := &usecase.DetachItemAddonUseCase{
			ItemAddonUseCaseDependencies: realDeps,
		}

		link, err := uc.Execute(ctx, detachInput(addonID))
		require.NoError(t, err)
		require.Equal(t, addonID, link.AddonID)

		_, err = getLink(addonID)
		require.ErrorIs(t, err, domain.ErrItemAddonNotFound)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemAddonDetached),
		))
	})

	t.Run("outbox payload retains composite identity", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		var aggregateID uuid.UUID
		var payload []byte

		err = tx.Raw().QueryRow(
			ctx, `
			SELECT aggregate_id, payload
			FROM outbox_events
			WHERE tenant_id = $1
			  AND aggregate_id = $2
			  AND event_type = $3
			ORDER BY occurred_at DESC
			LIMIT 1
		`,
			fixture.tenantID,
			fixture.itemID,
			string(domain.EventTypeItemAddonAttached),
		).Scan(&aggregateID, &payload)
		require.NoError(t, err)

		require.Equal(t, fixture.itemID, aggregateID)

		var event domain.ItemAddonEvent
		require.NoError(t, json.Unmarshal(payload, &event))
		require.Equal(t, fixture.itemID, event.Item)
		require.Equal(t, addonID, event.Addon)
		require.Equal(t, fixture.restaurantID, event.Restaurant)
	})
}
