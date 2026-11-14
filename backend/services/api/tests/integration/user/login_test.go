package user_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tenantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/repository"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	userusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
)

const integrationLoginPassword = "SecureLoginPassword123!"

func createLoginTestUser(
	t *testing.T,
	i *integrationDB,
	tenantID uuid.UUID,
	email string,
	password string,
) *userdomain.User {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	passwordHash, err := auth.HashPassword(password)
	require.NoError(t, err)

	user, err := userdomain.NewUser(
		tenantID,
		email,
		passwordHash,
		"Integration",
		userdomain.RoleOwner,
	)
	require.NoError(t, err)

	tx, err := i.db.BeginTx(ctx, tenantID)
	require.NoError(t, err)

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	repo := userrepository.NewPostgresUserRepository()

	require.NoError(t, repo.Create(ctx, tx, user))
	require.NoError(t, tx.Commit(ctx))

	return user
}

func newLoginTestUseCase(
	i *integrationDB,
) (*userusecase.LoginUseCase, *auth.TokenService) {
	tokenService, err := auth.NewTokenService(
		strings.Repeat("s", 32),
		"restaurantflow-integration-test",
		15*time.Minute,
	)
	if err != nil {
		panic(err)
	}

	login := &userusecase.LoginUseCase{
		DB:                    i.db,
		Tenants:               tenantrepository.NewPostgresTenantRepository(),
		Users:                 userrepository.NewPostgresUserRepository(),
		RefreshTokens:         userrepository.NewPostgresRefreshTokenRepository(),
		Tokens:                tokenService,
		RefreshTokenGenerator: auth.NewRefreshTokenGenerator(),
	}

	return login, tokenService
}

func TestIntegrationLoginUseCase(t *testing.T) {
	i := newIntegrationDB(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	tenantID := i.createTenant(t)
	slug := "user-it-" + tenantID.String()

	user := createLoginTestUser(
		t,
		i,
		tenantID,
		"owner@example.com",
		integrationLoginPassword,
	)

	login, tokenService := newLoginTestUseCase(i)

	t.Run("successful login", func(t *testing.T) {
		result, err := login.Execute(ctx, userusecase.LoginInput{
			BusinessSlug: slug,
			Email:        " OWNER@EXAMPLE.COM ",
			Password:     integrationLoginPassword,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Tenant)
		require.NotNil(t, result.User)

		require.Equal(t, tenantID, result.Tenant.ID)
		require.Equal(t, user.ID, result.User.ID)
		require.Equal(t, "owner@example.com", result.User.Email)
		require.Equal(t, "Bearer", result.TokenType)
		require.False(t, result.ExpiresAt.IsZero())
		require.NotEmpty(t, result.AccessToken)

		// Verify refresh-token issuance.
		require.NotEmpty(t, result.RefreshToken)
		require.False(t, result.RefreshTokenExpiresAt.IsZero())

		require.WithinDuration(
			t,
			time.Now().Add(30*24*time.Hour),
			result.RefreshTokenExpiresAt,
			2*time.Minute,
		)

		claims, err := tokenService.ValidateAccessToken(
			result.AccessToken,
		)

		require.NoError(t, err)
		require.Equal(t, user.ID.String(), claims.Subject)
		require.Equal(t, tenantID.String(), claims.TenantID)
		require.Equal(t, "owner", claims.Role)

		// The tokens must never expose password data.
		require.NotContains(t, result.AccessToken, user.PasswordHash)
		require.NotContains(t, result.AccessToken, integrationLoginPassword)
		require.NotContains(t, result.RefreshToken, integrationLoginPassword)

		// Verify last login and refresh-token persistence.
		tx, err := i.db.BeginTx(ctx, tenantID)
		require.NoError(t, err)

		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		stored, err := userrepository.NewPostgresUserRepository().
			GetByID(ctx, tx, tenantID, user.ID)

		require.NoError(t, err)
		require.NotNil(t, stored.LastLoginAt)

		// Verify only the SHA-256 hash is stored.
		refreshTokenHash, hashErr := auth.HashRefreshToken(result.RefreshToken)
		require.NoError(t, hashErr)

		storedRefreshToken, err := userrepository.NewPostgresRefreshTokenRepository().
			GetByHash(ctx, tx, tenantID, refreshTokenHash)

		require.NoError(t, err)
		require.NotNil(t, storedRefreshToken)
		require.Equal(t, tenantID, storedRefreshToken.TenantID)
		require.Equal(t, user.ID, storedRefreshToken.UserID)
		require.Equal(t, refreshTokenHash, storedRefreshToken.TokenHash)

		require.WithinDuration(
			t,
			result.RefreshTokenExpiresAt,
			storedRefreshToken.ExpiresAt,
			time.Second,
		)

		require.Nil(t, storedRefreshToken.RevokedAt)
		require.Nil(t, storedRefreshToken.ReplacedByTokenID)
	})

	t.Run("invalid credentials return same error", func(t *testing.T) {
		testCases := []struct {
			name  string
			input userusecase.LoginInput
		}{
			{
				name: "wrong password",
				input: userusecase.LoginInput{
					BusinessSlug: slug,
					Email:        "owner@example.com",
					Password:     "WrongPassword123!",
				},
			},
			{
				name: "unknown email",
				input: userusecase.LoginInput{
					BusinessSlug: slug,
					Email:        "missing@example.com",
					Password:     integrationLoginPassword,
				},
			},
			{
				name: "unknown tenant",
				input: userusecase.LoginInput{
					BusinessSlug: "missing-" + uuid.NewString(),
					Email:        "owner@example.com",
					Password:     integrationLoginPassword,
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := login.Execute(ctx, tc.input)

				require.Nil(t, result)
				require.ErrorIs(t, err, userusecase.ErrInvalidCredentials)
			})
		}
	})

	t.Run("inactive user cannot login", func(t *testing.T) {
		tx, err := i.db.BeginTx(ctx, tenantID)
		require.NoError(t, err)

		_, err = tx.Raw().Exec(
			ctx,
			"UPDATE users SET status = 'inactive' WHERE id = $1",
			user.ID,
		)
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))

		result, err := login.Execute(ctx, userusecase.LoginInput{
			BusinessSlug: slug,
			Email:        user.Email,
			Password:     integrationLoginPassword,
		})

		require.Nil(t, result)
		require.ErrorIs(t, err, userusecase.ErrInvalidCredentials)
	})
}

func TestIntegrationLoginTenantIsolation(t *testing.T) {
	i := newIntegrationDB(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	tenantA := i.createTenant(t)
	tenantB := i.createTenant(t)

	userA := createLoginTestUser(
		t,
		i,
		tenantA,
		"owner@example.com",
		integrationLoginPassword,
	)

	userB := createLoginTestUser(
		t,
		i,
		tenantB,
		"owner@example.com",
		integrationLoginPassword,
	)

	login, tokenService := newLoginTestUseCase(i)

	testCases := []struct {
		name     string
		tenantID uuid.UUID
		userID   uuid.UUID
		slug     string
	}{
		{
			name:     "tenant A",
			tenantID: tenantA,
			userID:   userA.ID,
			slug:     "user-it-" + tenantA.String(),
		},
		{
			name:     "tenant B",
			tenantID: tenantB,
			userID:   userB.ID,
			slug:     "user-it-" + tenantB.String(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := login.Execute(ctx, userusecase.LoginInput{
				BusinessSlug: tc.slug,
				Email:        "owner@example.com",
				Password:     integrationLoginPassword,
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.RefreshToken)

			claims, err := tokenService.ValidateAccessToken(
				result.AccessToken,
			)

			require.NoError(t, err)
			require.Equal(t, tc.userID.String(), claims.Subject)
			require.Equal(t, tc.tenantID.String(), claims.TenantID)

			// Verify the refresh token belongs to the correct tenant and user.
			tx, err := i.db.BeginTx(ctx, tc.tenantID)
			require.NoError(t, err)

			refreshTokenHash, hashErr := auth.HashRefreshToken(result.RefreshToken)
			require.NoError(t, hashErr)

			storedRefreshToken, err := userrepository.NewPostgresRefreshTokenRepository().
				GetByHash(ctx, tx, tc.tenantID, refreshTokenHash)

			require.NoError(t, err)
			require.Equal(t, tc.tenantID, storedRefreshToken.TenantID)
			require.Equal(t, tc.userID, storedRefreshToken.UserID)

			require.NoError(t, tx.Rollback(context.Background()))
		})
	}
}
