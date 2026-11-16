package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterAddonRoutes(
	mux *http.ServeMux,
	handler *AddonHandler,
	authMiddleware func(http.Handler) http.Handler,
) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/addons",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.CreateAddon))),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/addons",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.ListAddons))),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/addons/{addonID}",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.GetAddon))),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/addons/{addonID}",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.UpdateAddon))),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/menu/addons/{addonID}/deactivate",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.DeactivateAddon))),
	)
}
