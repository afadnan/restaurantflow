package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	handler *RestaurantHandler,
) {
	mux.Handle(
		"POST /api/v1/restaurants",
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.CreateRestaurant),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants",
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.ListRestaurants),
		),
	)

	mux.Handle(
		"GET /api/v1/restaurants/{restaurantID}",
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.GetRestaurant),
		),
	)

	mux.Handle(
		"PUT /api/v1/restaurants/{restaurantID}",
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.UpdateRestaurant),
		),
	)

	mux.Handle(
		"PATCH /api/v1/restaurants/{restaurantID}/deactivate",
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.DeactivateRestaurant),
		),
	)
}
