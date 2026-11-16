package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterItemRoutes(
	mux *http.ServeMux,
	handler *ItemHandler,
	authMiddleware func(http.Handler) http.Handler,
) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/items",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.CreateItem))),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.ListItems))),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items/{itemID}",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.GetItem))),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/items/{itemID}",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.UpdateItem))),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/deactivate",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.DeactivateItem))),
	)
}
