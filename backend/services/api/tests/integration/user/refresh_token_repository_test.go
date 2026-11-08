package user_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
)

func createRefreshTokenTestUser(
	t *testing.T,
	i *integrationDB,
	ctx context.Context,
	tenantID uuid.UUID,
) *domain.User {
	t.Helper()

	user := newTestUser(
		t,
		tenantID,
		"refresh-"+uuid.NewString()+"@example.com",
	)

	userRepo := repository.NewPostgresUserRepository()

	tx, err := i.db.BeginTx(ctx, tenantID)
	require.NoError(t, err)
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	require.NoError(t, userRepo.Create(ctx, tx, user))
	require.NoError(t, tx.Commit(ctx))

	return user
}

func newRefreshTokenFixture(
	t *testing.T,
	tenantID uuid.UUID,
	userID uuid.UUID,
	familyID uuid.UUID,
	seed string,
) *domain.RefreshToken {
	t.Helper()

	hash := sha256.Sum256([]byte(seed))
	now := time.Now().UTC()

	token, err := domain.NewRefreshToken(
		tenantID,
		userID,
		familyID,
		hash[:],
		now.Add(30*24*time.Hour),
		now,
	)
	require.NoError(t, err)

	return token
}

func TestIntegrationRefreshTokenRepository(t *testing.T) {
	i := newIntegrationDB(t)

	tenantA := i.createTenant(t)
	tenantB := i.createTenant(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	userA := createRefreshTokenTestUser(t, i, ctx, tenantA)

	repo := repository.NewPostgresRefreshTokenRepository()

	t.Run("create and retrieve by hash", func(t *testing.T) {
		familyID := uuid.New()

		token := newRefreshTokenFixture(
			t,
			tenantA,
			userA.ID,
			familyID,
			"refresh-token-create-"+uuid.NewString(),
		)

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, token))
		require.NoError(t, tx.Commit(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		found, err := repo.GetByHash(
			ctx,
			tx,
			tenantA,
			token.TokenHash,
		)

		require.NoError(t, err)
		require.Equal(t, token.ID, found.ID)
		require.Equal(t, token.TenantID, found.TenantID)
		require.Equal(t, token.UserID, found.UserID)
		require.Equal(t, token.FamilyID, found.FamilyID)
		require.Equal(t, token.TokenHash, found.TokenHash)
		require.WithinDuration(
			t,
			token.ExpiresAt,
			found.ExpiresAt,
			time.Second,
		)
	})

	t.Run("lock token by hash", func(t *testing.T) {
		token := newRefreshTokenFixture(
			t,
			tenantA,
			userA.ID,
			uuid.New(),
			"refresh-token-lock-"+uuid.NewString(),
		)

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, token))
		require.NoError(t, tx.Commit(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		found, err := repo.LockByHash(
			ctx,
			tx,
			tenantA,
			token.TokenHash,
		)

		require.NoError(t, err)
		require.Equal(t, token.ID, found.ID)
		require.Equal(t, token.TokenHash, found.TokenHash)
	})

	t.Run("rotate token", func(t *testing.T) {
		familyID := uuid.New()

		oldToken := newRefreshTokenFixture(
			t,
			tenantA,
			userA.ID,
			familyID,
			"refresh-token-old-"+uuid.NewString(),
		)

		newToken := newRefreshTokenFixture(
			t,
			tenantA,
			userA.ID,
			familyID,
			"refresh-token-new-"+uuid.NewString(),
		)

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, oldToken))
		require.NoError(t, repo.Create(ctx, tx, newToken))

		require.NoError(t, repo.Rotate(
			ctx,
			tx,
			tenantA,
			oldToken.ID,
			newToken.ID,
		))

		require.NoError(t, tx.Commit(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		found, err := repo.GetByHash(
			ctx,
			tx,
			tenantA,
			oldToken.TokenHash,
		)

		require.NoError(t, err)
		require.True(t, found.IsRevoked())
		require.NotNil(t, found.ReplacedByTokenID)
		require.Equal(t, newToken.ID, *found.ReplacedByTokenID)
	})

	t.Run("revoke entire token family", func(t *testing.T) {
		familyID := uuid.New()

		first := newRefreshTokenFixture(
			t,
			tenantA,
			userA.ID,
			familyID,
			"refresh-family-first-"+uuid.NewString(),
		)

		second := newRefreshTokenFixture(
			t,
			tenantA,
			userA.ID,
			familyID,
			"refresh-family-second-"+uuid.NewString(),
		)

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, first))
		require.NoError(t, repo.Create(ctx, tx, second))
		require.NoError(t, tx.Commit(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.RevokeFamily(
			ctx,
			tx,
			tenantA,
			familyID,
		))

		require.NoError(t, tx.Commit(ctx))

		tx, err = i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		firstFound, err := repo.GetByHash(
			ctx,
			tx,
			tenantA,
			first.TokenHash,
		)
		require.NoError(t, err)
		require.True(t, firstFound.IsRevoked())

		secondFound, err := repo.GetByHash(
			ctx,
			tx,
			tenantA,
			second.TokenHash,
		)
		require.NoError(t, err)
		require.True(t, secondFound.IsRevoked())
	})

	t.Run("tenant isolation prevents cross-tenant lookup", func(t *testing.T) {
		token := newRefreshTokenFixture(
			t,
			tenantA,
			userA.ID,
			uuid.New(),
			"refresh-token-isolation-"+uuid.NewString(),
		)

		tx, err := i.db.BeginTx(ctx, tenantA)
		require.NoError(t, err)

		require.NoError(t, repo.Create(ctx, tx, token))
		require.NoError(t, tx.Commit(ctx))

		// The transaction belongs to tenant B, but the query
		// deliberately requests tenant A's token.
		tx, err = i.db.BeginTx(ctx, tenantB)
		require.NoError(t, err)
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		found, err := repo.GetByHash(
			ctx,
			tx,
			tenantA,
			token.TokenHash,
		)

		require.Nil(t, found)
		require.ErrorIs(
			t,
			err,
			repository.ErrRefreshTokenNotFound,
		)
	})
}
