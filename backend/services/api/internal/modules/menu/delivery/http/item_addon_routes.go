package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterItemAddonRoutes(mux *http.ServeMux, handler *ItemAddonHandler) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons",
		middleware.TenantMiddleware(http.HandlerFunc(handler.AttachItemAddon)),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons",
		middleware.TenantMiddleware(http.HandlerFunc(handler.ListItemAddons)),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons/{addonID}",
		middleware.TenantMiddleware(http.HandlerFunc(handler.GetItemAddon)),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons/{addonID}",
		middleware.TenantMiddleware(http.HandlerFunc(handler.UpdateItemAddon)),
	)

	mux.Handle(
		"DELETE /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/addons/{addonID}",
		middleware.TenantMiddleware(http.HandlerFunc(handler.DetachItemAddon)),
	)
}
