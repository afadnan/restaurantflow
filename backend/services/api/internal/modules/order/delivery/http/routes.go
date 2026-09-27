package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	handler *OrderHandler,
	websocketHandler *Handler,
) {
	mux.Handle(
		"POST /api/v1/orders",
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.CreateOrder),
		),
	)

	mux.Handle(
		"PATCH /api/v1/orders/{orderID}/kds",
		middleware.TenantMiddleware(
			http.HandlerFunc(handler.UpdateKDSState),
		),
	)

	if websocketHandler != nil {
		mux.Handle(
			"GET /ws/v1/kds",
			middleware.TenantMiddleware(
				http.HandlerFunc(websocketHandler.KDSWebSocket),
			),
		)
	}
}
