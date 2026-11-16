package middleware

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
)

// Protected applies authentication, tenant validation, and permission
// authorization to a protected HTTP handler.
func Protected(
	authMiddleware func(http.Handler) http.Handler,
	authorizer *authorization.Authorizer,
	permission authorization.Permission,
	handler http.Handler,
) http.Handler {
	return authMiddleware(
		TenantMiddleware(
			RequirePermission(authorizer, permission)(handler),
		),
	)
}
