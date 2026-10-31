package httpdelivery

import "net/http"

// RegisterRoutes registers public owner registration endpoints.
//
// This endpoint must not use TenantMiddleware because the
// tenant does not exist before registration.
func RegisterRoutes(
	mux *http.ServeMux,
	handler *OwnerRegistrationHandler,
) {
	mux.HandleFunc(
		"POST /api/v1/auth/register/owner",
		handler.RegisterOwner,
	)
}
