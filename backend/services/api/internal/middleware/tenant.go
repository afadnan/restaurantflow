package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type tenantContextKey struct{}

var ErrTenantMissing = errors.New("tenant context missing")

var ErrTenantMismatch = errors.New("tenant mismatch between JWT and request")

var ErrInvalidTenant = errors.New("invalid tenant id")

// TenantFromContext retrieves the authenticated tenant ID.
//
// The tenant ID is stored as a UUID rather than a raw string so callers
// cannot accidentally use malformed tenant identifiers.
func TenantFromContext(ctx context.Context) (uuid.UUID, error) {
	value := ctx.Value(tenantContextKey{})

	tenantID, ok := value.(uuid.UUID)
	if !ok || tenantID == uuid.Nil {
		return uuid.Nil, ErrTenantMissing
	}

	return tenantID, nil
}

// WithTenant stores a validated tenant ID in the context.
func WithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenantID)
}

// TenantMiddleware extracts tenant_id from a verified JWT or X-Tenant-ID.
//
// Security rules:
//
//  1. JWT tenant_id wins.
//  2. If X-Tenant-ID is present together with JWT tenant_id, both must match.
//  3. If no JWT tenant exists, X-Tenant-ID may be used.
//  4. An invalid tenant identifier is rejected.
//  5. No tenant means no request proceeds.
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))

		var (
			jwtTenant    uuid.UUID
			hasJWTTenant bool
		)

		if token, ok := r.Context().Value(JWTClaimsKey{}).(*jwt.Token); ok &&
			token != nil &&
			token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				rawTenant, exists := claims["tenant_id"]
				if exists {
					value, ok := rawTenant.(string)
					if !ok {
						http.Error(w, "invalid tenant_id claim", http.StatusUnauthorized)
						return
					}

					parsed, err := uuid.Parse(value)
					if err != nil || parsed == uuid.Nil {
						http.Error(w, "invalid tenant_id claim", http.StatusUnauthorized)
						return
					}

					jwtTenant = parsed
					hasJWTTenant = true
				}
			}
		}

		var tenantID uuid.UUID

		switch {
		case hasJWTTenant && headerTenant != "":
			headerID, err := uuid.Parse(headerTenant)
			if err != nil || headerID == uuid.Nil {
				http.Error(w, "invalid tenant id", http.StatusBadRequest)
				return
			}

			if headerID != jwtTenant {
				http.Error(w, ErrTenantMismatch.Error(), http.StatusForbidden)
				return
			}

			tenantID = jwtTenant

		case hasJWTTenant:
			tenantID = jwtTenant

		case headerTenant != "":
			parsed, err := uuid.Parse(headerTenant)
			if err != nil || parsed == uuid.Nil {
				http.Error(w, ErrInvalidTenant.Error(), http.StatusBadRequest)
				return
			}

			tenantID = parsed

		default:
			http.Error(w, ErrTenantMissing.Error(), http.StatusUnauthorized)
			return
		}

		ctx := WithTenant(r.Context(), tenantID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// JWTClaimsKey is intentionally exported so the authentication middleware
// can place the verified JWT into the request context before TenantMiddleware.
type JWTClaimsKey struct{}
