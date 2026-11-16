package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTenantMiddleware(t *testing.T) {
	tokenService, _, tenantID, accessToken := newMiddlewareTestToken(t)

	otherTenantID := uuid.New()

	tests := []struct {
		name               string
		authorization      string
		tenantHeader       string
		expectedStatusCode int
		expectNext         bool
	}{
		{
			name:               "missing tenant header uses JWT tenant",
			authorization:      "Bearer " + accessToken,
			expectedStatusCode: http.StatusOK,
			expectNext:         true,
		},
		{
			name:               "matching tenant header is accepted",
			authorization:      "Bearer " + accessToken,
			tenantHeader:       tenantID.String(),
			expectedStatusCode: http.StatusOK,
			expectNext:         true,
		},
		{
			name:               "tenant header mismatch is forbidden",
			authorization:      "Bearer " + accessToken,
			tenantHeader:       otherTenantID.String(),
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "malformed tenant header is rejected",
			authorization:      "Bearer " + accessToken,
			tenantHeader:       "not-a-uuid",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "nil tenant header is rejected",
			authorization:      "Bearer " + accessToken,
			tenantHeader:       uuid.Nil.String(),
			expectedStatusCode: http.StatusBadRequest,
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

				gotTenantID, err := TenantFromContext(r.Context())
				require.NoError(t, err)
				require.Equal(t, tenantID, gotTenantID)

				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(tokenService)(
				TenantMiddleware(next),
			)

			request := httptest.NewRequest(
				http.MethodGet,
				"/test",
				nil,
			)

			if tt.authorization != "" {
				request.Header.Set(
					"Authorization",
					tt.authorization,
				)
			}

			if tt.tenantHeader != "" {
				request.Header.Set(
					"X-Tenant-ID",
					tt.tenantHeader,
				)
			}

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

func TestTenantMiddlewareWithoutAuthenticatedContext(t *testing.T) {
	nextCalled := false

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := TenantMiddleware(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.False(t, nextCalled)
}

func TestTenantFromContext(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		ctx := httptest.NewRequest(
			http.MethodGet,
			"/test",
			nil,
		).Context()

		tenantID, err := TenantFromContext(ctx)

		require.ErrorIs(t, err, ErrTenantMissing)
		require.Equal(t, uuid.Nil, tenantID)
	})

	t.Run("valid tenant", func(t *testing.T) {
		expectedTenantID := uuid.New()

		ctx := WithTenant(
			httptest.NewRequest(
				http.MethodGet,
				"/test",
				nil,
			).Context(),
			expectedTenantID,
		)

		tenantID, err := TenantFromContext(ctx)

		require.NoError(t, err)
		require.Equal(t, expectedTenantID, tenantID)
	})

	t.Run("nil tenant", func(t *testing.T) {
		ctx := WithTenant(
			httptest.NewRequest(
				http.MethodGet,
				"/test",
				nil,
			).Context(),
			uuid.Nil,
		)

		tenantID, err := TenantFromContext(ctx)

		require.ErrorIs(t, err, ErrTenantMissing)
		require.Equal(t, uuid.Nil, tenantID)
	})
}
