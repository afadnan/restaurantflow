package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/usecase"
)

type OrderHandler struct {
	CreateOrderUseCase    *usecase.CreateOrderUseCase
	UpdateKDSStateUseCase *usecase.UpdateKDSStateUseCase
}

func NewOrderHandler(
	createOrder *usecase.CreateOrderUseCase,
	updateKDSState *usecase.UpdateKDSStateUseCase,
) *OrderHandler {
	return &OrderHandler{
		CreateOrderUseCase:    createOrder,
		UpdateKDSStateUseCase: updateKDSState,
	}
}

type createOrderRequest struct {
	RestaurantID uuid.UUID  `json:"restaurant_id"`
	TableID      *uuid.UUID `json:"table_id"`
	CustomerID   *uuid.UUID `json:"customer_id"`

	OrderType     domain.OrderType     `json:"order_type"`
	PaymentStatus domain.PaymentStatus `json:"payment_status"`

	Currency       string `json:"currency"`
	TaxAmount      int64  `json:"tax_amount"`
	DiscountAmount int64  `json:"discount_amount"`
	ServiceFee     int64  `json:"service_fee"`

	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	Notes         string `json:"notes"`

	Items []createOrderItemInput `json:"items"`
}

type createOrderItemInput struct {
	ProductID uuid.UUID `json:"product_id"`
	Name      string    `json:"name"`
	Quantity  int32     `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
	Notes     string    `json:"notes"`
}

func (h *OrderHandler) CreateOrder(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			"tenant context required",
			http.StatusUnauthorized,
		)
		return
	}

	var request createOrderRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if request.RestaurantID == uuid.Nil {
		http.Error(
			w,
			"restaurant_id is required",
			http.StatusBadRequest,
		)
		return
	}

	items := make(
		[]usecase.CreateOrderItemInput,
		0,
		len(request.Items),
	)

	for _, item := range request.Items {
		items = append(
			items,
			usecase.CreateOrderItemInput{
				ProductID: item.ProductID,
				Name:      item.Name,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				Notes:     item.Notes,
			},
		)
	}

	order, err := h.CreateOrderUseCase.Execute(
		r.Context(),
		usecase.CreateOrderInput{
			TenantID:       tenantID,
			RestaurantID:   request.RestaurantID,
			TableID:        request.TableID,
			CustomerID:     request.CustomerID,
			OrderType:      request.OrderType,
			PaymentStatus:  request.PaymentStatus,
			Currency:       request.Currency,
			TaxAmount:      request.TaxAmount,
			DiscountAmount: request.DiscountAmount,
			ServiceFee:     request.ServiceFee,
			CustomerName:   request.CustomerName,
			CustomerPhone:  request.CustomerPhone,
			Notes:          request.Notes,
			Items:          items,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		order,
	)
}

type updateKDSStateRequest struct {
	State domain.KDSState `json:"state"`
}

func (h *OrderHandler) UpdateKDSState(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			"tenant context required",
			http.StatusUnauthorized,
		)
		return
	}

	orderID, err := uuid.Parse(
		r.PathValue("orderID"),
	)
	if err != nil || orderID == uuid.Nil {
		http.Error(
			w,
			"invalid order id",
			http.StatusBadRequest,
		)
		return
	}

	var request updateKDSStateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	err = h.UpdateKDSStateUseCase.Execute(
		r.Context(),
		usecase.UpdateKDSStateInput{
			TenantID: tenantID,
			OrderID:  orderID,
			State:    request.State,
		},
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeDomainError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, domain.ErrInvalidTenantID):
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)

	case errors.Is(err, domain.ErrInvalidOrderID),
		errors.Is(err, domain.ErrInvalidRestaurantID),
		errors.Is(err, domain.ErrInvalidOrderItems),
		errors.Is(err, domain.ErrInvalidQuantity),
		errors.Is(err, domain.ErrInvalidPrice),
		errors.Is(err, domain.ErrInvalidKDSState),
		errors.Is(err, domain.ErrInvalidStateTransition):
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

	case errors.Is(err, domain.ErrOrderNotFound),
		errors.Is(err, domain.ErrKDSNotFound):
		http.Error(
			w,
			err.Error(),
			http.StatusNotFound,
		)

	case errors.Is(err, domain.ErrInsufficientStock):
		http.Error(
			w,
			err.Error(),
			http.StatusConflict,
		)

	default:
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}
