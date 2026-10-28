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
	pool *pgxpool.Pool
	db   *postgres.DB
}

func newIntegrationDB(t *testing.T) *integrationDB {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer cancel()

	cfg, err := config.Load()
	require.NoError(t, err)

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))

	t.Cleanup(pool.Close)

	return &integrationDB{
		pool: pool,
		db:   postgres.NewDB(pool),
	}
}

func (i *integrationDB) createTenant(t *testing.T) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(), 10*time.Second,
	)
	defer cancel()

	id := uuid.New()

	_, err := i.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, $2, $3)
	`, id, "User Integration Tenant", "user-it-"+id.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cancel()

		_, err := i.pool.Exec(ctx,
			"DELETE FROM tenants WHERE id = $1", id)
		require.NoError(t, err)
	})

	return id
}

func newTestUser(t *testing.T, tenantID uuid.UUID, email string) *domain.User {
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
		context.Background(), 30*time.Second,
	)
	defer cancel()

	t.Run("create and retrieve user", func(t *testing.T) {
		user := newTestUser(t, tenantA, "FIRST@example.com")

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, user))
		require.NoError(t, tx.Commit(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		byID, err := repo.GetByID(ctx, tx, tenantA, user.ID)
		require.NoError(t, err)
		require.Equal(t, user.ID, byID.ID)

		byEmail, err := repo.GetByEmail(
			ctx, tx, tenantA, "FIRST@example.com",
		)
		require.NoError(t, err)
		require.Equal(t, user.ID, byEmail.ID)
		require.Equal(t, "first@example.com", byEmail.Email)
	})

	t.Run("duplicate email rejected", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		user := newTestUser(t, tenantA, "first@example.com")

		err = repo.Create(ctx, tx, user)
		require.ErrorIs(t, err, repository.ErrEmailExists)

		require.NoError(t, tx.Rollback(ctx))
	})

	t.Run("same email allowed across tenants", func(t *testing.T) {
		user := newTestUser(t, tenantB, "first@example.com")

		tx, err := i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, user))
		require.NoError(t, tx.Commit(ctx))
	})

	t.Run("cross tenant read rejected", func(t *testing.T) {
		txA, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		userA, err := repo.GetByEmail(
			ctx, txA, tenantA, "first@example.com",
		)
		require.NoError(t, err)
		require.NoError(t, txA.Rollback(ctx))

		txB, err := i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)
		defer func() { _ = txB.Rollback(ctx) }()

		_, err = repo.GetByID(ctx, txB, tenantB, userA.ID)
		require.ErrorIs(t, err, repository.ErrUserNotFound)
	})

	t.Run("transaction rollback", func(t *testing.T) {
		user := newTestUser(t, tenantA, "rollback@example.com")

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, user))
		require.NoError(t, tx.Rollback(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = repo.GetByID(ctx, tx, tenantA, user.ID)
		require.ErrorIs(t, err, repository.ErrUserNotFound)
	})

	t.Run("update last login", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		user, err := repo.GetByEmail(
			ctx, tx, tenantA, "first@example.com",
		)
		require.NoError(t, err)

		updated, err := repo.UpdateLastLogin(
			ctx, tx, tenantA, user.ID,
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

	user := newTestUser(t, tenantA, "rls@example.com")

	txA, err := i.db.BeginTx(ctx, tenantA)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, txA, user))
	require.NoError(t, txA.Commit(ctx))

	t.Run("own tenant can read user", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

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
		defer func() { _ = tx.Rollback(ctx) }()

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
		defer func() { _ = tx.Rollback(ctx) }()

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
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = tx.Raw().Exec(
			ctx, `
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
