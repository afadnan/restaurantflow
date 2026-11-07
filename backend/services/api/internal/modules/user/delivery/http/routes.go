package httpdelivery

import "net/http"

// RegisterRoutes registers public authentication endpoints.
//
// These endpoints must not use TenantMiddleware because login
// establishes the authenticated tenant context.
func RegisterRoutes(
	mux *http.ServeMux,
	registrationHandler *OwnerRegistrationHandler,
	loginHandler *OwnerLoginHandler,
) {
	mux.HandleFunc(
		"POST /api/v1/auth/register/owner",
		registrationHandler.RegisterOwner,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		loginHandler.Login,
	)
}
