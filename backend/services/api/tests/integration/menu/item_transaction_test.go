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

var errItemEventFailure = errors.New("injected item event failure")

// The actual PostgreSQL item write executes first.
// Returning an error here tests whether that write rolls back.
type failingItemEvents struct{}

func (failingItemEvents) Append(
	_ context.Context,
	_ sharedevents.Transaction,
	_ sharedevents.DomainEvent,
) error {
	return errItemEventFailure
}

func TestIntegrationMenuItemTransaction(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	// Resolve the category belonging to the fixture item.
	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)

	var categoryID uuid.UUID
	err = tx.Raw().QueryRow(ctx, `
		SELECT category_id
		FROM menu_items
		WHERE id = $1
	`, fixture.itemID).Scan(&categoryID)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	itemRepo := repository.NewPostgresItemRepository()

	dependencies := func(
		eventRepo sharedevents.EventRepository,
	) usecase.ItemUseCaseDependencies {
		return usecase.ItemUseCaseDependencies{
			DB:     integration.db,
			Items:  itemRepo,
			Events: eventRepo,
		}
	}

	realDeps := dependencies(events.NewPostgresEventRepository())
	failingDeps := dependencies(failingItemEvents{})

	getItem := func(t *testing.T) *domain.Item {
		t.Helper()

		uc := &usecase.GetItemUseCase{
			ItemUseCaseDependencies: realDeps,
		}

		item, err := uc.Execute(ctx, usecase.GetItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
		})
		require.NoError(t, err)
		return item
	}

	createInput := func(name string) usecase.CreateItemInput {
		return usecase.CreateItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			CategoryID:   categoryID,
			Name:         name,
			SKU:          uuid.NewString(),
			Price:        "150.00",
			Status:       domain.ItemStatusActive,
			IsAvailable:  true,
		}
	}

	updateInput := func(name string) usecase.UpdateItemInput {
		return usecase.UpdateItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
			CategoryID:   categoryID,
			Name:         name,
			Price:        "250.00",
			Status:       domain.ItemStatusActive,
			IsAvailable:  true,
		}
	}

	countItemsByName := func(t *testing.T, name string) int {
		t.Helper()

		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		var count int
		err = tx.Raw().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM menu_items
			WHERE tenant_id = $1
			  AND restaurant_id = $2
			  AND name = $3
		`, fixture.tenantID, fixture.restaurantID, name).Scan(&count)
		require.NoError(t, err)

		return count
	}

	t.Run("create commits item and event", func(t *testing.T) {
		uc := &usecase.CreateItemUseCase{
			ItemUseCaseDependencies: realDeps,
		}

		name := "Transaction Successful Create"
		item, err := uc.Execute(ctx, createInput(name))
		require.NoError(t, err)
		require.NotNil(t, item)

		require.Equal(t, 1, countItemsByName(t, name))
		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, item.ID,
			string(domain.EventTypeItemCreated),
		))
	})

	t.Run("create rolls back when event fails", func(t *testing.T) {
		uc := &usecase.CreateItemUseCase{
			ItemUseCaseDependencies: failingDeps,
		}

		name := "Transaction Failed Create"
		item, err := uc.Execute(ctx, createInput(name))

		require.ErrorIs(t, err, errItemEventFailure)
		require.Nil(t, item)
		require.Zero(t, countItemsByName(t, name))
	})

	t.Run("update commits item and event", func(t *testing.T) {
		uc := &usecase.UpdateItemUseCase{
			ItemUseCaseDependencies: realDeps,
		}

		name := "Transaction Successful Update"
		item, err := uc.Execute(ctx, updateInput(name))
		require.NoError(t, err)
		require.NotNil(t, item)

		stored := getItem(t)
		require.Equal(t, name, stored.Name)
		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemUpdated),
		))
	})

	t.Run("update rolls back when event fails", func(t *testing.T) {
		before := getItem(t)

		uc := &usecase.UpdateItemUseCase{
			ItemUseCaseDependencies: failingDeps,
		}

		item, err := uc.Execute(
			ctx,
			updateInput("Transaction Failed Update"),
		)

		require.ErrorIs(t, err, errItemEventFailure)
		require.Nil(t, item)

		after := getItem(t)
		require.Equal(t, before, after)

		// The successful update in the previous subtest
		// is the only committed update event.
		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemUpdated),
		))
	})

	t.Run("deactivate rolls back when event fails", func(t *testing.T) {
		before := getItem(t)

		uc := &usecase.DeactivateItemUseCase{
			ItemUseCaseDependencies: failingDeps,
		}

		item, err := uc.Execute(ctx, usecase.DeactivateItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
		})

		require.ErrorIs(t, err, errItemEventFailure)
		require.Nil(t, item)

		after := getItem(t)
		require.Equal(t, before, after)

		require.Zero(t, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemDeactivated),
		))
	})

	t.Run("invalid category creates no item or event", func(t *testing.T) {
		uc := &usecase.CreateItemUseCase{
			ItemUseCaseDependencies: realDeps,
		}

		name := "Transaction Invalid Category"
		input := createInput(name)
		input.CategoryID = uuid.New()

		item, err := uc.Execute(ctx, input)

		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
		require.Nil(t, item)
		require.Zero(t, countItemsByName(t, name))
	})

	t.Run("invalid category preserves existing item", func(t *testing.T) {
		before := getItem(t)

		uc := &usecase.UpdateItemUseCase{
			ItemUseCaseDependencies: realDeps,
		}

		input := updateInput("Invalid Category Update")
		input.CategoryID = uuid.New()

		item, err := uc.Execute(ctx, input)

		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
		require.Nil(t, item)

		after := getItem(t)
		require.Equal(t, before, after)

		require.Equal(t, 1, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(domain.EventTypeItemUpdated),
		))
	})
}
