package menu_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
)

func TestIntegrationMenuAddonRepository(t *testing.T) {
	integration := newVariantIntegration(t)
	fixture := integration.createFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := repository.NewPostgresAddonRepository()

	newAddon := func(name string) *domain.Addon {
		t.Helper()
		addon, err := domain.NewAddon(
			fixture.tenantID,
			fixture.restaurantID,
			name,
			"Integration test addon",
			"25.00",
			time.Now().UTC(),
		)
		require.NoError(t, err)
		return addon
	}

	addon := newAddon("Extra Cheese")

	t.Run("create and commit", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		require.NoError(t, repo.Create(ctx, tx, addon))
		require.NoError(t, tx.Commit(ctx))
	})

	t.Run("get and list", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		stored, err := repo.GetByID(
			ctx, tx, fixture.tenantID, fixture.restaurantID, addon.ID,
		)
		require.NoError(t, err)
		require.Equal(t, addon.ID, stored.ID)
		require.Equal(t, "Extra Cheese", stored.Name)
		require.Equal(t, "25.00", stored.Price)
		require.True(t, stored.IsActive)

		addons, err := repo.List(
			ctx, tx, fixture.tenantID, fixture.restaurantID, 50, 0,
		)
		require.NoError(t, err)
		require.Len(t, addons, 1)
		require.Equal(t, addon.ID, addons[0].ID)
	})

	t.Run("wrong restaurant cannot read", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = repo.GetByID(
			ctx, tx, fixture.tenantID, uuid.New(), addon.ID,
		)
		require.ErrorIs(t, err, domain.ErrAddonNotFound)
	})

	t.Run("other tenant cannot read", func(t *testing.T) {
		otherTenantID := uuid.New()

		_, err := integration.pool.Exec(ctx, `
			INSERT INTO tenants (id, name, slug)
			VALUES ($1, $2, $3)
		`, otherTenantID, "Addon Other Tenant",
			"addon-other-"+otherTenantID.String())
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

		tx, err := integration.db.BeginTx(ctx, otherTenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = repo.GetByID(
			ctx, tx, otherTenantID, fixture.restaurantID, addon.ID,
		)
		require.ErrorIs(t, err, domain.ErrAddonNotFound)

		var count int
		err = tx.Raw().QueryRow(ctx, `
			SELECT COUNT(*) FROM menu_addons WHERE id = $1
		`, addon.ID).Scan(&count)
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("update", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		current, err := repo.GetByID(
			ctx, tx, fixture.tenantID, fixture.restaurantID, addon.ID,
		)
		require.NoError(t, err)

		require.NoError(t, current.Update(
			fixture.tenantID,
			fixture.restaurantID,
			addon.ID,
			"Premium Cheese",
			"Updated description",
			"35.00",
			time.Now().UTC(),
		))

		updated, err := repo.Update(ctx, tx, current)
		require.NoError(t, err)
		require.Equal(t, "Premium Cheese", updated.Name)
		require.Equal(t, "35.00", updated.Price)
		require.NoError(t, tx.Commit(ctx))
	})

	t.Run("rollback discards create", func(t *testing.T) {
		temporary := newAddon("Rollback Addon")

		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, tx, temporary))
		require.NoError(t, tx.Rollback(ctx))

		verifyTx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = verifyTx.Rollback(ctx) }()

		_, err = repo.GetByID(
			ctx, verifyTx,
			fixture.tenantID, fixture.restaurantID, temporary.ID,
		)
		require.ErrorIs(t, err, domain.ErrAddonNotFound)
	})

	t.Run("deactivate and reject further update", func(t *testing.T) {
		tx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		deactivated, err := repo.Deactivate(
			ctx, tx,
			fixture.tenantID, fixture.restaurantID, addon.ID,
			time.Now().UTC(),
		)
		require.NoError(t, err)
		require.False(t, deactivated.IsActive)
		require.NoError(t, tx.Commit(ctx))

		verifyTx, err := integration.db.BeginTx(ctx, fixture.tenantID)
		require.NoError(t, err)
		defer func() { _ = verifyTx.Rollback(ctx) }()

		deactivated.Name = "Should Not Update"
		_, err = repo.Update(ctx, verifyTx, deactivated)
		require.ErrorIs(t, err, domain.ErrAddonNotFound)
	})
}
