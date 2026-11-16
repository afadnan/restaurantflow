package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterAddonRoutes(
	mux *http.ServeMux,
	handler *AddonHandler,
	authMiddleware func(http.Handler) http.Handler,
	authorizer *authorization.Authorizer,
) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/addons",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuManageAddons,
			http.HandlerFunc(handler.CreateAddon),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/addons",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.ListAddons),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/addons/{addonID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.GetAddon),
		),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/addons/{addonID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuManageAddons,
			http.HandlerFunc(handler.UpdateAddon),
		),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/menu/addons/{addonID}/deactivate",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuManageAddons,
			http.HandlerFunc(handler.DeactivateAddon),
		),
	)
}
