package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type tenantContextKey struct{}

var (
	ErrTenantMissing  = errors.New("tenant context missing")
	ErrTenantMismatch = errors.New("tenant mismatch between JWT and request")
	ErrInvalidTenant  = errors.New("invalid tenant id")
)

func TenantFromContext(ctx context.Context) (uuid.UUID, error) {
	value := ctx.Value(tenantContextKey{})

	tenantID, ok := value.(uuid.UUID)
	if !ok || tenantID == uuid.Nil {
		return uuid.Nil, ErrTenantMissing
	}

	return tenantID, nil
}

func WithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenantID)
}

// TenantMiddleware requires an authenticated tenant identity.
// X-Tenant-ID is optional and can only confirm the JWT tenant, never select one.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtTenantID, err := AuthenticatedTenantFromContext(r.Context())
		if err != nil {
			http.Error(w, ErrAuthenticationRequired.Error(), http.StatusUnauthorized)
			return
		}

		headerTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))

		if headerTenant != "" {
			headerTenantID, parseErr := uuid.Parse(headerTenant)
			if parseErr != nil || headerTenantID == uuid.Nil {
				http.Error(w, ErrInvalidTenant.Error(), http.StatusBadRequest)
				return
			}

			if headerTenantID != jwtTenantID {
				http.Error(w, ErrTenantMismatch.Error(), http.StatusForbidden)
				return
			}
		}

		ctx := WithTenant(r.Context(), jwtTenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
