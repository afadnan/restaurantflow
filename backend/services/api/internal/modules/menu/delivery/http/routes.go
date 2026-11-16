package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterCategoryRoutes(
	mux *http.ServeMux,
	handler *CategoryHandler,
	authMiddleware func(http.Handler) http.Handler,
	authorizer *authorization.Authorizer,
) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/categories",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuCreate,
			http.HandlerFunc(handler.CreateCategory),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/categories",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.ListCategories),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/categories/{categoryID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.GetCategory),
		),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/categories/{categoryID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuUpdate,
			http.HandlerFunc(handler.UpdateCategory),
		),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/menu/categories/{categoryID}/deactivate",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuDelete,
			http.HandlerFunc(handler.DeactivateCategory),
		),
	)
}
