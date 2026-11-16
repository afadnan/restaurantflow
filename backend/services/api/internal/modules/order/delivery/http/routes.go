package httpdelivery

import (
	"net/http"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func RegisterRoutes(
	mux *http.ServeMux,
	handler *OrderHandler,
	websocketHandler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	mux.Handle(
		"POST /api/v1/orders",
		authMiddleware(
			middleware.TenantMiddleware(
				http.HandlerFunc(handler.CreateOrder),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/orders/{orderID}/kds",
		authMiddleware(
			middleware.TenantMiddleware(
				http.HandlerFunc(handler.UpdateKDSState),
			),
		),
	)

	if websocketHandler != nil {
		mux.Handle(
			"GET /ws/v1/kds",
			authMiddleware(
				middleware.TenantMiddleware(
					http.HandlerFunc(websocketHandler.KDSWebSocket),
				),
			),
		)
	}
}
