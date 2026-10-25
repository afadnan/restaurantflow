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

func TestIntegrationMenuCategoryTenantAndRestaurantIsolation(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	repo := menurepository.NewPostgresCategoryRepository()
	eventRepo := events.NewPostgresEventRepository()

	deps := menuusecase.CategoryUseCaseDependencies{
		DB:         integration.db,
		Categories: repo,
		Events:     eventRepo,
	}

	getUC := &menuusecase.GetCategoryUseCase{
		CategoryUseCaseDependencies: deps,
	}
	listUC := &menuusecase.ListCategoriesUseCase{
		CategoryUseCaseDependencies: deps,
	}
	updateUC := &menuusecase.UpdateCategoryUseCase{
		CategoryUseCaseDependencies: deps,
	}
	deactivateUC := &menuusecase.DeactivateCategoryUseCase{
		CategoryUseCaseDependencies: deps,
	}

	var categoryID uuid.UUID

	tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)

	err = tx.Raw().QueryRow(ctx, `
		SELECT id
		FROM menu_categories
		WHERE tenant_id = $1
		  AND restaurant_id = $2
		LIMIT 1
	`, fixture.tenantID, fixture.restaurantID).Scan(&categoryID)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback(ctx))

	otherTenantID := uuid.New()

	_, err = integration.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, $2, $3)
	`, otherTenantID,
		"Category Isolation Tenant",
		"category-isolation-"+otherTenantID.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cleanupCancel()

		_, _ = integration.pool.Exec(
			cleanupCtx,
			`DELETE FROM tenants WHERE id = $1`,
			otherTenantID,
		)
	})

	otherRestaurantID := uuid.New()

	tx, err = integration.db.BeginTx(ctx, fixture.tenantID)
	require.NoError(t, err)

	_, err = tx.Raw().Exec(ctx, `
		INSERT INTO restaurants (id, tenant_id, name, slug)
		VALUES ($1, $2, $3, $4)
	`, otherRestaurantID, fixture.tenantID,
		"Other Category Restaurant",
		"other-category-"+otherRestaurantID.String())
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	t.Run("owner can read category", func(t *testing.T) {
		category, err := getUC.Execute(ctx, menuusecase.GetCategoryInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			CategoryID:   categoryID,
		})

		require.NoError(t, err)
		require.Equal(t, categoryID, category.ID)
	})

	t.Run("other tenant cannot read category", func(t *testing.T) {
		_, err := getUC.Execute(ctx, menuusecase.GetCategoryInput{
			TenantID:     otherTenantID,
			RestaurantID: fixture.restaurantID,
			CategoryID:   categoryID,
		})

		require.ErrorIs(t, err, menudomain.ErrCategoryNotFound)
	})

	t.Run("other restaurant cannot read category", func(t *testing.T) {
		_, err := getUC.Execute(ctx, menuusecase.GetCategoryInput{
			TenantID:     fixture.tenantID,
			RestaurantID: otherRestaurantID,
			CategoryID:   categoryID,
		})

		require.ErrorIs(t, err, menudomain.ErrCategoryNotFound)
	})

	t.Run("other tenant cannot list category", func(t *testing.T) {
		categories, err := listUC.Execute(ctx, menuusecase.ListCategoriesInput{
			TenantID:     otherTenantID,
			RestaurantID: fixture.restaurantID,
		})

		require.NoError(t, err)
		require.Empty(t, categories)
	})

	t.Run("other restaurant cannot list category", func(t *testing.T) {
		categories, err := listUC.Execute(ctx, menuusecase.ListCategoriesInput{
			TenantID:     fixture.tenantID,
			RestaurantID: otherRestaurantID,
		})

		require.NoError(t, err)
		require.Empty(t, categories)
	})

	t.Run("other tenant cannot update category", func(t *testing.T) {
		_, err := updateUC.Execute(ctx, menuusecase.UpdateCategoryInput{
			TenantID:     otherTenantID,
			RestaurantID: fixture.restaurantID,
			CategoryID:   categoryID,
			Name:         "Unauthorized Update",
		})

		require.ErrorIs(t, err, menudomain.ErrCategoryNotFound)
	})

	t.Run("other restaurant cannot deactivate category", func(t *testing.T) {
		_, err := deactivateUC.Execute(ctx, menuusecase.DeactivateCategoryInput{
			TenantID:     fixture.tenantID,
			RestaurantID: otherRestaurantID,
			CategoryID:   categoryID,
		})

		require.ErrorIs(t, err, menudomain.ErrCategoryNotFound)
	})

	t.Run("RLS hides category from other tenant", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, otherTenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		var count int

		err = tx.Raw().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM menu_categories
			WHERE id = $1
		`, categoryID).Scan(&count)

		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("unauthorized attempts do not mutate category", func(t *testing.T) {
		category, err := getUC.Execute(ctx, menuusecase.GetCategoryInput{
			TenantID:     fixture.tenantID,
			RestaurantID: fixture.restaurantID,
			CategoryID:   categoryID,
		})

		require.NoError(t, err)
		require.Equal(t, "Variant Integration Category", category.Name)
		require.True(t, category.IsActive)

		require.Zero(t, integration.countEvents(
			t,
			fixture.tenantID,
			categoryID,
			string(menudomain.EventTypeCategoryUpdated),
		))

		require.Zero(t, integration.countEvents(
			t,
			fixture.tenantID,
			categoryID,
			string(menudomain.EventTypeCategoryDeactivated),
		))
	})
}
