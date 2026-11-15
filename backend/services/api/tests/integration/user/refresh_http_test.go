package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	userhttp "github.com/afadnan/restaurantflow/services/api/internal/modules/user/delivery/http"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	userusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
)

func TestIntegrationRefreshHTTP(t *testing.T) {
	i := newIntegrationDB(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	tenantID := i.createTenant(t)
	slug := "refresh-it-" + tenantID.String()

	user := createLoginTestUser(
		t,
		i,
		tenantID,
		"refresh-owner@example.com",
		integrationLoginPassword,
	)

	loginUseCase, tokenService := newLoginTestUseCase(i)

	loginResult, err := loginUseCase.Execute(
		ctx,
		userusecase.LoginInput{
			BusinessSlug: slug,
			Email:        "refresh-owner@example.com",
			Password:     integrationLoginPassword,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, loginResult)
	require.NotEmpty(t, loginResult.RefreshToken)

	refreshUseCase := &userusecase.RefreshUseCase{
		DB:                    i.db,
		Users:                 userrepository.NewPostgresUserRepository(),
		RefreshTokens:         userrepository.NewPostgresRefreshTokenRepository(),
		Tokens:                tokenService,
		RefreshTokenGenerator: auth.NewRefreshTokenGenerator(),
	}

	refreshHandler := userhttp.NewRefreshHandler(refreshUseCase)

	mux := http.NewServeMux()
	userhttp.RegisterRoutes(mux, nil, nil, refreshHandler)

	sendRefreshRequest := func(
		requestTenantID string,
		refreshToken string,
	) *httptest.ResponseRecorder {
		t.Helper()

		body, err := json.Marshal(map[string]string{
			"tenant_id":     requestTenantID,
			"refresh_token": refreshToken,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/refresh",
			bytes.NewReader(body),
		).WithContext(ctx)

		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		return recorder
	}

	// ---------------------------------------------------------
	// Test 1: Successful refresh and token rotation
	// ---------------------------------------------------------

	t.Run("successful refresh rotates refresh token", func(t *testing.T) {
		recorder := sendRefreshRequest(
			tenantID.String(),
			loginResult.RefreshToken,
		)

		require.Equal(
			t,
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)

		require.Equal(
			t,
			"application/json",
			recorder.Header().Get("Content-Type"),
		)

		var response struct {
			AccessToken           string    `json:"access_token"`
			TokenType             string    `json:"token_type"`
			ExpiresAt             time.Time `json:"expires_at"`
			RefreshToken          string    `json:"refresh_token"`
			RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`

			User struct {
				ID        string `json:"id"`
				TenantID  string `json:"tenant_id"`
				Email     string `json:"email"`
				FirstName string `json:"first_name"`
				Role      string `json:"role"`
				Status    string `json:"status"`
			} `json:"user"`
		}

		require.NoError(
			t,
			json.Unmarshal(recorder.Body.Bytes(), &response),
		)

		require.NotEmpty(t, response.AccessToken)
		require.Equal(t, "Bearer", response.TokenType)
		require.False(t, response.ExpiresAt.IsZero())

		require.NotEmpty(t, response.RefreshToken)
		require.NotEqual(
			t,
			loginResult.RefreshToken,
			response.RefreshToken,
		)
		require.False(t, response.RefreshTokenExpiresAt.IsZero())

		require.Equal(t, user.ID.String(), response.User.ID)
		require.Equal(t, tenantID.String(), response.User.TenantID)
		require.Equal(t, user.Email, response.User.Email)
		require.Equal(t, "owner", response.User.Role)
		require.Equal(t, "active", response.User.Status)

		claims, err := tokenService.ValidateAccessToken(
			response.AccessToken,
		)
		require.NoError(t, err)
		require.Equal(t, user.ID.String(), claims.Subject)
		require.Equal(t, tenantID.String(), claims.TenantID)
		require.Equal(t, "owner", claims.Role)

		// The old refresh token must now be revoked and point
		// to the replacement token.
		oldHash, err := auth.HashRefreshToken(
			loginResult.RefreshToken,
		)
		require.NoError(t, err)

		newHash, err := auth.HashRefreshToken(
			response.RefreshToken,
		)
		require.NoError(t, err)

		tx, err := i.db.BeginTx(ctx, tenantID)
		require.NoError(t, err)

		oldStored, err := userrepository.
			NewPostgresRefreshTokenRepository().
			GetByHash(ctx, tx, tenantID, oldHash)
		require.NoError(t, err)
		require.NotNil(t, oldStored)
		require.NotNil(t, oldStored.RevokedAt)
		require.NotNil(t, oldStored.ReplacedByTokenID)

		newStored, err := userrepository.
			NewPostgresRefreshTokenRepository().
			GetByHash(ctx, tx, tenantID, newHash)
		require.NoError(t, err)
		require.NotNil(t, newStored)
		require.Nil(t, newStored.RevokedAt)

		require.Equal(
			t,
			oldStored.FamilyID,
			newStored.FamilyID,
		)
		require.Equal(
			t,
			newStored.ID,
			*oldStored.ReplacedByTokenID,
		)

		require.NoError(t, tx.Rollback(ctx))

		// Reusing the old token must be rejected.
		replayRecorder := sendRefreshRequest(
			tenantID.String(),
			loginResult.RefreshToken,
		)

		require.Equal(
			t,
			http.StatusUnauthorized,
			replayRecorder.Code,
			replayRecorder.Body.String(),
		)

		// Reuse detection must revoke the replacement token too.
		replacementRecorder := sendRefreshRequest(
			tenantID.String(),
			response.RefreshToken,
		)

		require.Equal(
			t,
			http.StatusUnauthorized,
			replacementRecorder.Code,
			replacementRecorder.Body.String(),
		)
	})

	// ---------------------------------------------------------
	// Test 2: Unknown refresh token is rejected
	// ---------------------------------------------------------

	t.Run("unknown refresh token rejected", func(t *testing.T) {
		recorder := sendRefreshRequest(
			tenantID.String(),
			"unknown-refresh-token",
		)

		require.Equal(
			t,
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	})

	// ---------------------------------------------------------
	// Test 3: Expired refresh token is rejected
	// ---------------------------------------------------------

	t.Run("expired refresh token rejected", func(t *testing.T) {
		generated, err := auth.NewRefreshTokenGenerator().Generate()
		require.NoError(t, err)
		require.NotNil(t, generated)

		now := time.Now().UTC()
		createdAt := now.Add(-2 * time.Hour)
		expiresAt := now.Add(-time.Hour)

		expiredToken, err := userdomain.NewRefreshToken(
			tenantID,
			user.ID,
			uuid.New(),
			generated.Hash,
			expiresAt,
			createdAt,
		)
		require.NoError(t, err)

		tx, err := i.db.BeginTx(ctx, tenantID)
		require.NoError(t, err)

		err = userrepository.
			NewPostgresRefreshTokenRepository().
			Create(ctx, tx, expiredToken)
		require.NoError(t, err)

		require.NoError(t, tx.Commit(ctx))

		recorder := sendRefreshRequest(
			tenantID.String(),
			generated.Plaintext,
		)

		require.Equal(
			t,
			http.StatusUnauthorized,
			recorder.Code,
			recorder.Body.String(),
		)
	})

	// ---------------------------------------------------------
	// Test 4: Invalid tenant ID is rejected
	// ---------------------------------------------------------

	t.Run("invalid tenant ID rejected", func(t *testing.T) {
		recorder := sendRefreshRequest(
			"not-a-valid-uuid",
			"some-refresh-token",
		)

		require.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	})

	// ---------------------------------------------------------
	// Test 5: Malformed JSON is rejected
	// ---------------------------------------------------------

	t.Run("malformed JSON rejected", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/refresh",
			bytes.NewBufferString(`{"tenant_id":`),
		).WithContext(ctx)

		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		require.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	})

	// ---------------------------------------------------------
	// Test 6: Unknown JSON fields are rejected
	// ---------------------------------------------------------

	t.Run("unknown JSON field rejected", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"tenant_id":     tenantID.String(),
			"refresh_token": "some-refresh-token",
			"admin":         true,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/refresh",
			bytes.NewReader(body),
		).WithContext(ctx)

		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		require.Equal(
			t,
			http.StatusBadRequest,
			recorder.Code,
			recorder.Body.String(),
		)
	})
}
