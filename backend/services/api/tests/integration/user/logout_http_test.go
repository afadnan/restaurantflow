package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	userhttp "github.com/afadnan/restaurantflow/services/api/internal/modules/user/delivery/http"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	userusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
)

func TestIntegrationLogoutHTTP(t *testing.T) {
	i := newIntegrationDB(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	tenantID := i.createTenant(t)
	slug := "user-it-" + tenantID.String()
	email := "logout-owner@example.com"

	createLoginTestUser(
		t,
		i,
		tenantID,
		email,
		integrationLoginPassword,
	)

	loginUseCase, tokenService := newLoginTestUseCase(i)

	login := func() *userusecase.LoginResult {
		t.Helper()

		result, err := loginUseCase.Execute(
			ctx,
			userusecase.LoginInput{
				BusinessSlug: slug,
				Email:        email,
				Password:     integrationLoginPassword,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.RefreshToken)

		return result
	}

	sessionA := login()
	sessionB := login()

	require.NotEqual(t, sessionA.RefreshToken, sessionB.RefreshToken)

	refreshUseCase := &userusecase.RefreshUseCase{
		DB:                    i.db,
		Users:                 userrepository.NewPostgresUserRepository(),
		RefreshTokens:         userrepository.NewPostgresRefreshTokenRepository(),
		Tokens:                tokenService,
		RefreshTokenGenerator: auth.NewRefreshTokenGenerator(),
	}

	logoutUseCase := &userusecase.LogoutUseCase{
		DB:            i.db,
		RefreshTokens: userrepository.NewPostgresRefreshTokenRepository(),
	}

	mux := http.NewServeMux()
	userhttp.RegisterRoutes(
		mux,
		nil,
		nil,
		userhttp.NewRefreshHandler(refreshUseCase),
		userhttp.NewLogoutHandler(logoutUseCase),
	)

	sendRequest := func(
		path string,
		refreshToken string,
	) *httptest.ResponseRecorder {
		t.Helper()

		body, err := json.Marshal(map[string]string{
			"tenant_id":     tenantID.String(),
			"refresh_token": refreshToken,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			path,
			bytes.NewReader(body),
		).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)

		return recorder
	}

	t.Run("logout revokes only the presented session", func(t *testing.T) {
		logoutRecorder := sendRequest(
			"/api/v1/auth/logout",
			sessionA.RefreshToken,
		)

		require.Equal(
			t,
			http.StatusNoContent,
			logoutRecorder.Code,
			logoutRecorder.Body.String(),
		)
		require.Empty(t, logoutRecorder.Body.String())

		// The logged-out session must no longer be refreshable.
		revokedSessionRecorder := sendRequest(
			"/api/v1/auth/refresh",
			sessionA.RefreshToken,
		)

		require.Equal(
			t,
			http.StatusUnauthorized,
			revokedSessionRecorder.Code,
			revokedSessionRecorder.Body.String(),
		)

		// Logout is idempotent for an already-revoked token.
		repeatedLogoutRecorder := sendRequest(
			"/api/v1/auth/logout",
			sessionA.RefreshToken,
		)

		require.Equal(
			t,
			http.StatusNoContent,
			repeatedLogoutRecorder.Code,
			repeatedLogoutRecorder.Body.String(),
		)

		// Unknown tokens also receive the same successful logout response.
		unknownTokenRecorder := sendRequest(
			"/api/v1/auth/logout",
			"unknown-refresh-token",
		)

		require.Equal(
			t,
			http.StatusNoContent,
			unknownTokenRecorder.Code,
			unknownTokenRecorder.Body.String(),
		)

		// A separate login session belonging to the same user remains valid.
		otherSessionRecorder := sendRequest(
			"/api/v1/auth/refresh",
			sessionB.RefreshToken,
		)

		require.Equal(
			t,
			http.StatusOK,
			otherSessionRecorder.Code,
			otherSessionRecorder.Body.String(),
		)
	})

	t.Run("invalid tenant ID is rejected", func(t *testing.T) {
		body, err := json.Marshal(map[string]string{
			"tenant_id":     "not-a-valid-uuid",
			"refresh_token": "some-refresh-token",
		})
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/auth/logout",
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
