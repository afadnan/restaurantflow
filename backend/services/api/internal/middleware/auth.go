package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	platformauth "github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
	"github.com/google/uuid"
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrInvalidAuthentication  = errors.New("invalid authentication")
	ErrUserMissing            = errors.New("user context missing")
	ErrRoleMissing            = errors.New("role context missing")
)

type authenticatedUser struct {
	userID   uuid.UUID
	tenantID uuid.UUID
	role     string
}

type authenticatedUserContextKey struct{}

// AuthMiddleware validates access tokens and stores the authenticated
// user's identity in the request context.
func AuthMiddleware(
	tokenService *platformauth.TokenService,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			authorization := strings.TrimSpace(
				r.Header.Get("Authorization"),
			)

			if authorization == "" {
				http.Error(
					w,
					ErrAuthenticationRequired.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			parts := strings.Fields(authorization)

			if len(parts) != 2 ||
				!strings.EqualFold(parts[0], "Bearer") {
				http.Error(
					w,
					ErrInvalidAuthentication.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			claims, err := tokenService.ValidateAccessToken(parts[1])
			if err != nil {
				http.Error(
					w,
					ErrInvalidAuthentication.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil || userID == uuid.Nil {
				http.Error(
					w,
					ErrInvalidAuthentication.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			tenantID, err := uuid.Parse(claims.TenantID)
			if err != nil || tenantID == uuid.Nil {
				http.Error(
					w,
					ErrInvalidAuthentication.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			user := authenticatedUser{
				userID:   userID,
				tenantID: tenantID,
				role:     strings.TrimSpace(claims.Role),
			}

			ctx := context.WithValue(
				r.Context(),
				authenticatedUserContextKey{},
				user,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext retrieves the authenticated user's UUID.
func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	user, ok := authenticatedUserFromContext(ctx)
	if !ok || user.userID == uuid.Nil {
		return uuid.Nil, ErrUserMissing
	}

	return user.userID, nil
}

// AuthenticatedTenantFromContext retrieves the tenant UUID
// from the verified JWT identity.
func AuthenticatedTenantFromContext(
	ctx context.Context,
) (uuid.UUID, error) {
	user, ok := authenticatedUserFromContext(ctx)
	if !ok || user.tenantID == uuid.Nil {
		return uuid.Nil, ErrTenantMissing
	}

	return user.tenantID, nil
}

// RoleFromContext retrieves the authenticated user's role.
func RoleFromContext(ctx context.Context) (string, error) {
	user, ok := authenticatedUserFromContext(ctx)
	if !ok || strings.TrimSpace(user.role) == "" {
		return "", ErrRoleMissing
	}

	return user.role, nil
}

func authenticatedUserFromContext(
	ctx context.Context,
) (authenticatedUser, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey{}).(authenticatedUser)
	return user, ok
}
