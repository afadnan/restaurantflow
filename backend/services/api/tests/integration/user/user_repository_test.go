package user_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/config"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/database"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

type integrationDB struct {
	pool        *pgxpool.Pool
	cleanupPool *pgxpool.Pool
	db          *postgres.DB
}

func newIntegrationDB(t *testing.T) *integrationDB {
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

	// Prevent integration tests from accessing the
	// development or production database.
	if cfg.Database.Name != "restaurantflow_test" {
		t.Fatalf(
			"integration tests require restaurantflow_test database; got %q",
			cfg.Database.Name,
		)
	}

	// Application database connection.
	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx))

	// Dedicated test-only cleanup connection.
	// This must NOT call newIntegrationDB(t).
	cleanupPool := newOwnerCleanupPool(t)

	return &integrationDB{
		pool:        pool,
		cleanupPool: cleanupPool,
		db:          postgres.NewDB(pool),
	}
}

func (i *integrationDB) createTenant(t *testing.T) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	id := uuid.New()

	_, err := i.pool.Exec(
		ctx, `
		INSERT INTO public.tenants (
			id,
			name,
			slug
		)
		VALUES ($1, $2, $3)
	`,
		id,
		"User Integration Tenant",
		"user-it-"+id.String(),
	)
	require.NoError(t, err)

	// The application role must not be used for
	// destructive integration-test cleanup.
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cleanupCancel()

		tx, err := i.cleanupPool.Begin(cleanupCtx)
		if err != nil {
			t.Errorf("begin tenant cleanup: %v", err)
			return
		}

		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		// Set the tenant context within this transaction.
		// PostgreSQL RLS on users requires this setting.
		_, err = tx.Exec(
			cleanupCtx,
			"SELECT set_config('app.tenant_id', $1, true)",
			id.String(),
		)
		if err != nil {
			t.Errorf("set cleanup tenant context: %v", err)
			return
		}

		// Only delete users belonging to the exact
		// tenant created by this test.
		_, err = tx.Exec(
			cleanupCtx,
			"DELETE FROM public.users WHERE tenant_id = $1",
			id,
		)
		if err != nil {
			t.Errorf("delete test users: %v", err)
			return
		}

		// Delete the exact test-created tenant.
		result, err := tx.Exec(
			cleanupCtx,
			"DELETE FROM public.tenants WHERE id = $1",
			id,
		)
		if err != nil {
			t.Errorf("delete test tenant: %v", err)
			return
		}

		if result.RowsAffected() != 1 {
			t.Errorf(
				"expected one test tenant deleted, got %d",
				result.RowsAffected(),
			)
			return
		}

		if err := tx.Commit(cleanupCtx); err != nil {
			t.Errorf("commit tenant cleanup: %v", err)
			return
		}

		// Verify the tenant was removed.
		var remaining int

		err = i.cleanupPool.QueryRow(
			cleanupCtx,
			"SELECT COUNT(*) FROM public.tenants WHERE id = $1",
			id,
		).Scan(&remaining)
		if err != nil {
			t.Errorf("verify tenant cleanup: %v", err)
			return
		}

		if remaining != 0 {
			t.Errorf(
				"tenant cleanup incomplete: %d rows remain",
				remaining,
			)
		}
	})

	return id
}

func newTestUser(
	t *testing.T,
	tenantID uuid.UUID,
	email string,
) *domain.User {
	t.Helper()

	user, err := domain.NewUser(
		tenantID,
		email,
		"test-hash",
		"Integration",
		domain.RoleStaff,
	)
	require.NoError(t, err)

	return user
}

func TestIntegrationUserRepository(t *testing.T) {
	i := newIntegrationDB(t)

	tenantA := i.createTenant(t)
	tenantB := i.createTenant(t)

	repo := repository.NewPostgresUserRepository()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	t.Run("create and retrieve user", func(t *testing.T) {
		user := newTestUser(
			t,
			tenantA,
			"FIRST@example.com",
		)

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, user))
		require.NoError(t, tx.Commit(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		byID, err := repo.GetByID(
			ctx,
			tx,
			tenantA,
			user.ID,
		)
		require.NoError(t, err)
		require.Equal(t, user.ID, byID.ID)

		byEmail, err := repo.GetByEmail(
			ctx,
			tx,
			tenantA,
			"FIRST@example.com",
		)
		require.NoError(t, err)

		require.Equal(t, user.ID, byEmail.ID)
		require.Equal(t, "first@example.com", byEmail.Email)
	})

	t.Run("duplicate email rejected", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		user := newTestUser(
			t,
			tenantA,
			"first@example.com",
		)

		err = repo.Create(ctx, tx, user)
		require.ErrorIs(t, err, repository.ErrEmailExists)
	})

	t.Run("same email allowed across tenants", func(t *testing.T) {
		user := newTestUser(
			t,
			tenantB,
			"first@example.com",
		)

		tx, err := i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, user))
		require.NoError(t, tx.Commit(ctx))
	})

	t.Run("cross tenant read rejected", func(t *testing.T) {
		txA, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		userA, err := repo.GetByEmail(
			ctx,
			txA,
			tenantA,
			"first@example.com",
		)
		require.NoError(t, err)
		require.NoError(t, txA.Rollback(ctx))

		txB, err := i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)
		defer func() {
			_ = txB.Rollback(context.Background())
		}()

		_, err = repo.GetByID(
			ctx,
			txB,
			tenantB,
			userA.ID,
		)
		require.ErrorIs(t, err, repository.ErrUserNotFound)
	})

	t.Run("transaction rollback", func(t *testing.T) {
		user := newTestUser(
			t,
			tenantA,
			"rollback@example.com",
		)

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, user))
		require.NoError(t, tx.Rollback(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		_, err = repo.GetByID(
			ctx,
			tx,
			tenantA,
			user.ID,
		)
		require.ErrorIs(t, err, repository.ErrUserNotFound)
	})

	t.Run("update last login", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		user, err := repo.GetByEmail(
			ctx,
			tx,
			tenantA,
			"first@example.com",
		)
		require.NoError(t, err)

		updated, err := repo.UpdateLastLogin(
			ctx,
			tx,
			tenantA,
			user.ID,
		)
		require.NoError(t, err)
		require.NotNil(t, updated.LastLoginAt)
	})
}

func TestIntegrationUserRLS(t *testing.T) {
	i := newIntegrationDB(t)

	tenantA := i.createTenant(t)
	tenantB := i.createTenant(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	repo := repository.NewPostgresUserRepository()

	user := newTestUser(
		t,
		tenantA,
		"rls@example.com",
	)

	txA, err := i.db.BeginTx(ctx, tenantA)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, txA, user))
	require.NoError(t, txA.Commit(ctx))

	t.Run("own tenant can read user", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		var count int

		err = tx.Raw().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM users
			WHERE id = $1
		`, user.ID).Scan(&count)

		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("other tenant cannot read user", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		var count int

		err = tx.Raw().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM users
			WHERE id = $1
		`, user.ID).Scan(&count)

		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("other tenant cannot update user", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		tag, err := tx.Raw().Exec(ctx, `
			UPDATE users
			SET first_name = 'Unauthorized'
			WHERE id = $1
		`, user.ID)

		require.NoError(t, err)
		require.EqualValues(t, 0, tag.RowsAffected())
	})

	t.Run("other tenant cannot insert foreign tenant user", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		_, err = tx.Raw().Exec(
			ctx,
			`
			INSERT INTO users (
				id,
				tenant_id,
				email,
				password_hash,
				first_name,
				role,
				status
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			`,
			uuid.New(),
			tenantA,
			"foreign@example.com",
			"test-hash",
			"Foreign",
			"staff",
			"active",
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "row-level security")
	})
}
