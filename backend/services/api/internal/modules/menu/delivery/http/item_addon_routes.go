package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterItemAddonRoutes(
	mux *http.ServeMux,
	handler *ItemAddonHandler,
	authMiddleware func(http.Handler) http.Handler,
	authorizer *authorization.Authorizer,
) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuManageAddons,
			http.HandlerFunc(handler.AttachItemAddon),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.ListItemAddons),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons/{addonID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.GetItemAddon),
		),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons/{addonID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuManageAddons,
			http.HandlerFunc(handler.UpdateItemAddon),
		),
	)

	mux.Handle(
		"DELETE /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons/{addonID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuManageAddons,
			http.HandlerFunc(handler.DetachItemAddon),
		),
	)
}
