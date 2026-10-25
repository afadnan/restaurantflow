package menu_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/events"
)

func TestIntegrationMenuItemCategoryOwnership(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deps := usecase.ItemUseCaseDependencies{
		DB:     integration.db,
		Items:  repository.NewPostgresItemRepository(),
		Events: events.NewPostgresEventRepository(),
	}

	createUC := &usecase.CreateItemUseCase{
		ItemUseCaseDependencies: deps,
	}
	getUC := &usecase.GetItemUseCase{
		ItemUseCaseDependencies: deps,
	}
	updateUC := &usecase.UpdateItemUseCase{
		ItemUseCaseDependencies: deps,
	}

	// Retrieve the category created by the existing fixture.
	var ownCategoryID uuid.UUID

	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)

	err = tx.Raw().QueryRow(ctx, `
		SELECT category_id
		FROM menu_items
		WHERE id = $1
	`, fixture.itemID).Scan(&ownCategoryID)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	// Create another restaurant and category under the same tenant.
	otherRestaurantID := uuid.New()
	otherCategoryID := uuid.New()

	tx, err = integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO restaurants (id, tenant_id, name, slug)
		VALUES ($1, $2, $3, $4)
	`, otherRestaurantID, fixture.tenantID,
		"Ownership Test Restaurant",
		"ownership-"+otherRestaurantID.String())
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO menu_categories (
			id, tenant_id, restaurant_id, name
		)
		VALUES ($1, $2, $3, $4)
	`, otherCategoryID, fixture.tenantID,
		otherRestaurantID, "Other Restaurant Category")
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	// Create a second tenant with its own restaurant and category.
	otherTenantID := uuid.New()
	otherTenantRestaurantID := uuid.New()
	otherTenantCategoryID := uuid.New()

	_, err = integration.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, $2, $3)
	`, otherTenantID, "Ownership Test Tenant",
		"ownership-tenant-"+otherTenantID.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cleanupCancel()

		// The test tenant has no items or outbox events.
		_, _ = integration.pool.Exec(
			cleanupCtx,
			`DELETE FROM tenants WHERE id = $1`,
			otherTenantID,
		)
	})

	tx, err = integration.db.BeginTx(ctx, otherTenantID)
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO restaurants (id, tenant_id, name, slug)
		VALUES ($1, $2, $3, $4)
	`, otherTenantRestaurantID, otherTenantID,
		"Other Tenant Restaurant",
		"other-tenant-"+otherTenantRestaurantID.String())
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO menu_categories (
			id, tenant_id, restaurant_id, name
		)
		VALUES ($1, $2, $3, $4)
	`, otherTenantCategoryID, otherTenantID,
		otherTenantRestaurantID, "Other Tenant Category")
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	newItemInput := func(categoryID uuid.UUID) usecase.CreateItemInput {
		return usecase.CreateItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			CategoryID:   categoryID,
			Name:         "Ownership Test Item",
			SKU:          "OWNERSHIP-" + uuid.NewString(),
			Price:        "150.00",
			Status:       domain.ItemStatusActive,
			IsAvailable:  true,
		}
	}

	t.Run("valid category permits item creation", func(t *testing.T) {
		item, err := createUC.Execute(ctx, newItemInput(ownCategoryID))

		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, ownCategoryID, item.CategoryID)
	})

	t.Run("other restaurant category rejects creation", func(t *testing.T) {
		_, err := createUC.Execute(ctx, newItemInput(otherCategoryID))
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("other tenant category rejects creation", func(t *testing.T) {
		_, err := createUC.Execute(ctx, newItemInput(otherTenantCategoryID))
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("other restaurant category rejects update", func(t *testing.T) {
		_, err := updateUC.Execute(ctx, usecase.UpdateItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
			CategoryID:   otherCategoryID,
			Name:         "Unauthorized Category Change",
			Price:        "200.00",
			Status:       domain.ItemStatusActive,
			IsAvailable:  true,
		})

		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("other tenant category rejects update", func(t *testing.T) {
		_, err := updateUC.Execute(ctx, usecase.UpdateItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
			CategoryID:   otherTenantCategoryID,
			Name:         "Unauthorized Tenant Category",
			Price:        "200.00",
			Status:       domain.ItemStatusActive,
			IsAvailable:  true,
		})

		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("failed updates preserve original category", func(t *testing.T) {
		item, err := getUC.Execute(ctx, usecase.GetItemInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			ItemID:       fixture.itemID,
		})

		require.NoError(t, err)
		require.Equal(t, ownCategoryID, item.CategoryID)
		require.Equal(t, "Variant Integration Item", item.Name)
	})
}
