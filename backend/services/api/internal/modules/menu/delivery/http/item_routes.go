package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterItemRoutes(mux *http.ServeMux, handler *ItemHandler) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/items",
		middleware.TenantMiddleware(http.HandlerFunc(handler.CreateItem)),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items",
		middleware.TenantMiddleware(http.HandlerFunc(handler.ListItems)),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items/{itemID}",
		middleware.TenantMiddleware(http.HandlerFunc(handler.GetItem)),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/items/{itemID}",
		middleware.TenantMiddleware(http.HandlerFunc(handler.UpdateItem)),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/deactivate",
		middleware.TenantMiddleware(http.HandlerFunc(handler.DeactivateItem)),
	)
}
