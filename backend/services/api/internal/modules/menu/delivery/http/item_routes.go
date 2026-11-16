package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterItemRoutes(
	mux *http.ServeMux,
	handler *ItemHandler,
	authMiddleware func(http.Handler) http.Handler,
	authorizer *authorization.Authorizer,
) {
	mux.Handle(
		"POST /api/v1/restaurants/{restaurantID}/menu/items",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuCreate,
			http.HandlerFunc(handler.CreateItem),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.ListItems),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}/menu/items/{itemID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuRead,
			http.HandlerFunc(handler.GetItem),
		),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}/menu/items/{itemID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuUpdate,
			http.HandlerFunc(handler.UpdateItem),
		),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/menu/items/{itemID}/deactivate",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionMenuDelete,
			http.HandlerFunc(handler.DeactivateItem),
		),
	)
}
