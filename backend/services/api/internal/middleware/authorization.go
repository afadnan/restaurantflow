package middleware

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
)

const ErrPermissionDenied = "permission denied"

// RequirePermission checks whether the authenticated user's role
// has the permission required by the route.
//
// This middleware must run after AuthMiddleware.
func RequirePermission(
	authorizer *authorization.Authorizer,
	permission authorization.Permission,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			role, err := RoleFromContext(r.Context())
			if err != nil {
				http.Error(
					w,
					ErrAuthenticationRequired.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			if authorizer == nil ||
				!authorizer.HasPermission(
					userdomain.Role(role),
					permission,
				) {
				http.Error(
					w,
					ErrPermissionDenied,
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
