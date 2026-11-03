package tenant_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

func newIntegrationDB(t *testing.T) *postgres.DB {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	cfg, err := config.Load()
	require.NoError(t, err)

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	return postgres.NewDB(pool)
}

func TestIntegrationTenantRepository(t *testing.T) {
	i := newIntegrationDB(t)
	repo := repository.NewPostgresTenantRepository()

	t.Run("create tenant", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		tenant, err := domain.NewTenant(
			"Integration Restaurant",
			"tenant-it-"+uuid.NewString(),
		)
		require.NoError(t, err)

		tx, err := i.BeginTx(ctx, tenant.ID)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		require.NoError(t, repo.Create(ctx, tx, tenant))
		require.Equal(t, "active", tenant.Status)
		require.False(t, tenant.CreatedAt.IsZero())
		require.False(t, tenant.UpdatedAt.IsZero())

		var count int

		err = tx.Raw().QueryRow(
			ctx,
			"SELECT COUNT(*) FROM tenants WHERE id = $1",
			tenant.ID,
		).Scan(&count)

		require.NoError(t, err)
		require.Equal(t, 1, count)

		require.NoError(t, tx.Rollback(ctx))
	})

	t.Run("duplicate slug rejected", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		slug := "tenant-duplicate-" + uuid.NewString()

		first, err := domain.NewTenant("First Restaurant", slug)
		require.NoError(t, err)

		second, err := domain.NewTenant("Second Restaurant", slug)
		require.NoError(t, err)

		tx, err := i.BeginTx(ctx, first.ID)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		require.NoError(t, repo.Create(ctx, tx, first))

		// A duplicate slug must be rejected by the unique constraint.
		// Use a savepoint because PostgreSQL marks the transaction
		// as failed after a constraint violation.
		_, err = tx.Raw().Exec(ctx, "SAVEPOINT duplicate_slug")
		require.NoError(t, err)

		err = repo.Create(ctx, tx, second)
		require.ErrorIs(t, err, repository.ErrTenantSlugExists)

		_, err = tx.Raw().Exec(
			ctx,
			"ROLLBACK TO SAVEPOINT duplicate_slug",
		)
		require.NoError(t, err)

		require.NoError(t, tx.Rollback(ctx))
	})

	t.Run("transaction rollback", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		tenant, err := domain.NewTenant(
			"Rollback Restaurant",
			"tenant-rollback-"+uuid.NewString(),
		)
		require.NoError(t, err)

		tx, err := i.BeginTx(ctx, tenant.ID)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		require.NoError(t, repo.Create(ctx, tx, tenant))
		require.NoError(t, tx.Rollback(ctx))

		var count int

		err = i.Pool.QueryRow(
			ctx,
			"SELECT COUNT(*) FROM tenants WHERE id = $1",
			tenant.ID,
		).Scan(&count)

		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("get tenant by slug", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		tenant, err := domain.NewTenant(
			"Lookup Integration Restaurant",
			"tenant-lookup-"+uuid.NewString(),
		)
		require.NoError(t, err)

		tx, err := i.BeginTx(ctx, tenant.ID)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		require.NoError(t, repo.Create(ctx, tx, tenant))
		require.NoError(t, tx.Commit(ctx))

		found, err := repo.GetBySlug(ctx, i, tenant.Slug)

		require.NoError(t, err)
		require.Equal(t, tenant.ID, found.ID)
		require.Equal(t, tenant.Name, found.Name)
		require.Equal(t, tenant.Slug, found.Slug)
		require.Equal(t, tenant.Status, found.Status)
	})

	t.Run("get tenant by normalized slug", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		tenant, err := domain.NewTenant(
			"Normalized Lookup Restaurant",
			"tenant-normalized-"+uuid.NewString(),
		)
		require.NoError(t, err)

		tx, err := i.BeginTx(ctx, tenant.ID)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		require.NoError(t, repo.Create(ctx, tx, tenant))
		require.NoError(t, tx.Commit(ctx))

		found, err := repo.GetBySlug(
			ctx,
			i,
			"  "+strings.ToUpper(tenant.Slug)+"  ",
		)

		require.NoError(t, err)
		require.Equal(t, tenant.ID, found.ID)
		require.Equal(t, tenant.Name, found.Name)
		require.Equal(t, tenant.Slug, found.Slug)
	})

	t.Run("unknown slug returns not found", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		_, err := repo.GetBySlug(
			ctx,
			i,
			"tenant-missing-"+uuid.NewString(),
		)

		require.ErrorIs(t, err, repository.ErrTenantNotFound)
	})
}
