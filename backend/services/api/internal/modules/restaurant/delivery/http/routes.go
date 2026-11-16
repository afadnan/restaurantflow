package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/authorization"
	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	handler *RestaurantHandler,
	authMiddleware func(http.Handler) http.Handler,
	authorizer *authorization.Authorizer,
) {
	mux.Handle(
		"POST /api/v1/restaurants",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionRestaurantCreate,
			http.HandlerFunc(handler.CreateRestaurant),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionRestaurantRead,
			http.HandlerFunc(handler.ListRestaurants),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionRestaurantRead,
			http.HandlerFunc(handler.GetRestaurant),
		),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionRestaurantUpdateProfile,
			http.HandlerFunc(handler.UpdateRestaurant),
		),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/deactivate",
		middleware.Protected(
			authMiddleware,
			authorizer,
			authorization.PermissionRestaurantDeactivate,
			http.HandlerFunc(handler.DeactivateRestaurant),
		),
	)
}
