package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	platformauth "github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const middlewareTestSecret = "restaurantflow-middleware-integration-test-secret"

func newMiddlewareTestToken(
	t *testing.T,
) (*platformauth.TokenService, uuid.UUID, uuid.UUID, string) {
	t.Helper()

	tokenService, err := platformauth.NewTokenService(
		middlewareTestSecret,
		"restaurantflow-middleware-test",
		time.Hour,
	)
	require.NoError(t, err)

	userID := uuid.New()
	tenantID := uuid.New()

	accessToken, _, err := tokenService.GenerateAccessToken(
		userID,
		tenantID,
		"owner",
	)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)

	return tokenService, userID, tenantID, accessToken
}

func TestAuthMiddleware(t *testing.T) {
	tokenService, userID, tenantID, accessToken := newMiddlewareTestToken(t)

	tests := []struct {
		name               string
		authorization      string
		expectedStatusCode int
		expectNext         bool
	}{
		{
			name:               "missing authorization header",
			authorization:      "",
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "missing bearer scheme",
			authorization:      accessToken,
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "unsupported authorization scheme",
			authorization:      "Basic " + accessToken,
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "malformed authorization header",
			authorization:      "Bearer",
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "extra authorization fields",
			authorization:      "Bearer " + accessToken + " extra",
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "invalid access token",
			authorization:      "Bearer invalid-token",
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "valid access token",
			authorization:      "Bearer " + accessToken,
			expectedStatusCode: http.StatusOK,
			expectNext:         true,
		},
		{
			name:               "bearer scheme is case insensitive",
			authorization:      "bearer " + accessToken,
			expectedStatusCode: http.StatusOK,
			expectNext:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false

			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				nextCalled = true

				gotUserID, err := UserIDFromContext(r.Context())
				require.NoError(t, err)
				require.Equal(t, userID, gotUserID)

				gotTenantID, err := AuthenticatedTenantFromContext(r.Context())
				require.NoError(t, err)
				require.Equal(t, tenantID, gotTenantID)

				gotRole, err := RoleFromContext(r.Context())
				require.NoError(t, err)
				require.Equal(t, "owner", gotRole)

				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(tokenService)(next)

			request := httptest.NewRequest(
				http.MethodGet,
				"/test",
				nil,
			)
			request.Header.Set("Authorization", tt.authorization)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			require.Equal(
				t,
				tt.expectedStatusCode,
				response.Code,
			)
			require.Equal(t, tt.expectNext, nextCalled)
		})
	}
}

func TestContextHelpersWithoutAuthenticatedUser(t *testing.T) {
	ctx := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	).Context()

	t.Run("user ID is missing", func(t *testing.T) {
		userID, err := UserIDFromContext(ctx)

		require.ErrorIs(t, err, ErrUserMissing)
		require.Equal(t, uuid.Nil, userID)
	})

	t.Run("authenticated tenant is missing", func(t *testing.T) {
		tenantID, err := AuthenticatedTenantFromContext(ctx)

		require.ErrorIs(t, err, ErrTenantMissing)
		require.Equal(t, uuid.Nil, tenantID)
	})

	t.Run("role is missing", func(t *testing.T) {
		role, err := RoleFromContext(ctx)

		require.ErrorIs(t, err, ErrRoleMissing)
		require.Empty(t, role)
	})
}
