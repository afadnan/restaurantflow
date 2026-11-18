package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeAuthenticatedUserValidator struct {
	role      string
	err       error
	calls     int
	gotUser   uuid.UUID
	gotTenant uuid.UUID
}

func (f *fakeAuthenticatedUserValidator) ValidateAuthenticatedUser(
	_ context.Context,
	tenantID uuid.UUID,
	userID uuid.UUID,
) (string, error) {
	f.calls++
	f.gotUser = userID
	f.gotTenant = tenantID

	return f.role, f.err
}

func TestAuthenticatedUserMiddleware(t *testing.T) {
	tokenService, userID, tenantID, accessToken := newMiddlewareTestToken(t)

	tests := []struct {
		name               string
		validator          AuthenticatedUserValidator
		expectedStatusCode int
		expectedRole       string
		expectNext         bool
	}{
		{
			name: "current database role replaces JWT role",
			validator: &fakeAuthenticatedUserValidator{
				role: "kitchen",
			},
			expectedStatusCode: http.StatusOK,
			expectedRole:       "kitchen",
			expectNext:         true,
		},
		{
			name: "inactive or missing user is unauthorized",
			validator: &fakeAuthenticatedUserValidator{
				err: ErrInvalidAuthenticatedUser,
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name: "database failure returns service unavailable",
			validator: &fakeAuthenticatedUserValidator{
				err: errors.New("database connection failed"),
			},
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			name:               "nil validator returns service unavailable",
			validator:          nil,
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			name: "empty current role is unauthorized",
			validator: &fakeAuthenticatedUserValidator{
				role: "",
			},
			expectedStatusCode: http.StatusUnauthorized,
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

				role, err := RoleFromContext(r.Context())
				require.NoError(t, err)
				require.Equal(t, tt.expectedRole, role)

				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(tokenService)(
				AuthenticatedUserMiddleware(tt.validator)(next),
			)

			request := httptest.NewRequest(
				http.MethodGet,
				"/test",
				nil,
			)
			request.Header.Set("Authorization", "Bearer "+accessToken)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			require.Equal(t, tt.expectedStatusCode, response.Code)
			require.Equal(t, tt.expectNext, nextCalled)

			if validator, ok := tt.validator.(*fakeAuthenticatedUserValidator); ok {
				require.Equal(t, 1, validator.calls)
				require.Equal(t, userID, validator.gotUser)
				require.Equal(t, tenantID, validator.gotTenant)
			}
		})
	}
}

func TestAuthenticatedUserMiddlewareWithoutJWTIdentity(t *testing.T) {
	validator := &fakeAuthenticatedUserValidator{
		role: "owner",
	}

	nextCalled := false

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthenticatedUserMiddleware(validator)(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.False(t, nextCalled)
	require.Zero(t, validator.calls)
}

func TestProtectedPreventsPrivilegeEscalationFromJWTRole(t *testing.T) {
	tokenService, userID, tenantID, accessToken := newMiddlewareTestToken(t)

	// The signed JWT claims owner, but the current database role is staff.
	validator := &fakeAuthenticatedUserValidator{
		role: "staff",
	}

	authMiddleware := func(next http.Handler) http.Handler {
		return AuthMiddleware(tokenService)(
			AuthenticatedUserMiddleware(validator)(next),
		)
	}

	handlerCalled := false

	protectedHandler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		handlerCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := Protected(
		authMiddleware,
		authorization.NewAuthorizer(),
		authorization.PermissionRestaurantDelete,
		protectedHandler,
	)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/restaurants/current",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.False(t, handlerCalled)

	require.Equal(t, 1, validator.calls)
	require.Equal(t, userID, validator.gotUser)
	require.Equal(t, tenantID, validator.gotTenant)
}
