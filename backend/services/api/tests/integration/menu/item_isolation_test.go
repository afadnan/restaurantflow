package menu_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	menudomain "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	menurepository "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	menuusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
)

func TestIntegrationMenuItemTenantAndRestaurantIsolation(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	deps := menuusecase.ItemUseCaseDependencies{
		DB:     integration.db,
		Items:  menurepository.NewPostgresItemRepository(),
		Events: events.NewPostgresEventRepository(),
	}

	getUC := &menuusecase.GetItemUseCase{ItemUseCaseDependencies: deps}
	listUC := &menuusecase.ListItemsUseCase{ItemUseCaseDependencies: deps}
	updateUC := &menuusecase.UpdateItemUseCase{ItemUseCaseDependencies: deps}
	deactivateUC := &menuusecase.DeactivateItemUseCase{ItemUseCaseDependencies: deps}

	otherTenantID := uuid.New()
	_, err := integration.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, $2, $3)
	`, otherTenantID, "Item Isolation Tenant",
		"item-isolation-"+otherTenantID.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := integration.pool.Exec(
			cleanupCtx, `DELETE FROM tenants WHERE id = $1`, otherTenantID,
		)
		require.NoError(t, err)
	})

	otherRestaurantID := uuid.New()
	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO restaurants (id, tenant_id, name, slug)
		VALUES ($1, $2, $3, $4)
	`, otherRestaurantID, fixture.tenantID,
		"Item Isolation Restaurant",
		"item-restaurant-"+otherRestaurantID.String())
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	t.Run("owner can read item", func(t *testing.T) {
		item, err := getUC.Execute(ctx, menuusecase.GetItemInput{
			TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
			ItemID: fixture.itemID,
		})
		require.NoError(t, err)
		require.Equal(t, fixture.itemID, item.ID)
	})

	t.Run("other tenant cannot read item", func(t *testing.T) {
		_, err := getUC.Execute(ctx, menuusecase.GetItemInput{
			TenantID: otherTenantID, RestaurantID: fixture.restaurantID,
			ItemID: fixture.itemID,
		})
		require.ErrorIs(t, err, menudomain.ErrItemNotFound)
	})

	t.Run("other restaurant cannot read item", func(t *testing.T) {
		_, err := getUC.Execute(ctx, menuusecase.GetItemInput{
			TenantID: fixture.tenantID, RestaurantID: otherRestaurantID,
			ItemID: fixture.itemID,
		})
		require.ErrorIs(t, err, menudomain.ErrItemNotFound)
	})

	t.Run("other tenant cannot list item", func(t *testing.T) {
		items, err := listUC.Execute(ctx, menuusecase.ListItemsInput{
			TenantID: otherTenantID, RestaurantID: fixture.restaurantID,
		})
		require.NoError(t, err)
		require.Empty(t, items)
	})

	t.Run("other restaurant cannot list item", func(t *testing.T) {
		items, err := listUC.Execute(ctx, menuusecase.ListItemsInput{
			TenantID: fixture.tenantID, RestaurantID: otherRestaurantID,
		})
		require.NoError(t, err)
		require.Empty(t, items)
	})

	t.Run("other tenant cannot update item", func(t *testing.T) {
		_, err := updateUC.Execute(ctx, menuusecase.UpdateItemInput{
			TenantID: otherTenantID, RestaurantID: fixture.restaurantID,
			ItemID: fixture.itemID,
		})
		require.ErrorIs(t, err, menudomain.ErrItemNotFound)
	})

	t.Run("other restaurant cannot deactivate item", func(t *testing.T) {
		_, err := deactivateUC.Execute(ctx, menuusecase.DeactivateItemInput{
			TenantID: fixture.tenantID, RestaurantID: otherRestaurantID,
			ItemID: fixture.itemID,
		})
		require.ErrorIs(t, err, menudomain.ErrItemNotFound)
	})

	t.Run("RLS hides item from other tenant", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, otherTenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		var count int
		err = tx.Raw().QueryRow(ctx, `
			SELECT COUNT(*) FROM menu_items WHERE id = $1
		`, fixture.itemID).Scan(&count)
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("unauthorized attempts do not mutate item", func(t *testing.T) {
		item, err := getUC.Execute(ctx, menuusecase.GetItemInput{
			TenantID: fixture.tenantID, RestaurantID: fixture.restaurantID,
			ItemID: fixture.itemID,
		})
		require.NoError(t, err)
		require.Equal(t, "Variant Integration Item", item.Name)

		require.Zero(t, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(menudomain.EventTypeItemUpdated),
		))
		require.Zero(t, integration.countEvents(
			t, fixture.tenantID, fixture.itemID,
			string(menudomain.EventTypeItemDeactivated),
		))
	})
}
