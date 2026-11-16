package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterCategoryRoutes(
	mux *http.ServeMux,
	handler *CategoryHandler,
	authMiddleware func(http.Handler) http.Handler,
) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/categories",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.CreateCategory))),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/categories",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.ListCategories))),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/categories/{categoryID}",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.GetCategory))),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/categories/{categoryID}",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.UpdateCategory))),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/menu/categories/{categoryID}/deactivate",
		authMiddleware(middleware.TenantMiddleware(http.HandlerFunc(handler.DeactivateCategory))),
	)
}
