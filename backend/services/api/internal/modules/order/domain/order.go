package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrderState string

const (
	OrderStatePending   OrderState = "pending"
	OrderStateConfirmed OrderState = "confirmed"
	OrderStatePreparing OrderState = "preparing"
	OrderStateReady     OrderState = "ready"
	OrderStateCompleted OrderState = "completed"
	OrderStateCancelled OrderState = "cancelled"
)

func (s OrderState) Valid() bool {
	switch s {
	case OrderStatePending,
		OrderStateConfirmed,
		OrderStatePreparing,
		OrderStateReady,
		OrderStateCompleted,
		OrderStateCancelled:
		return true
	default:
		return false
	}
}

func (s OrderState) CanTransitionTo(next OrderState) bool {
	switch s {
	case OrderStatePending:
		return next == OrderStateConfirmed ||
			next == OrderStatePreparing ||
			next == OrderStateCancelled

	case OrderStateConfirmed:
		return next == OrderStatePreparing ||
			next == OrderStateCancelled

	case OrderStatePreparing:
		return next == OrderStateReady ||
			next == OrderStateCancelled

	case OrderStateReady:
		return next == OrderStateCompleted ||
			next == OrderStateCancelled

	case OrderStateCompleted,
		OrderStateCancelled:
		return false

	default:
		return false
	}
}

type OrderType string

const (
	OrderTypeDineIn   OrderType = "dine_in"
	OrderTypeTakeaway OrderType = "takeaway"
	OrderTypeDelivery OrderType = "delivery"
)

func (t OrderType) Valid() bool {
	switch t {
	case OrderTypeDineIn,
		OrderTypeTakeaway,
		OrderTypeDelivery:
		return true
	default:
		return false
	}
}

type PaymentStatus string

const (
	PaymentStatusPending     PaymentStatus = "pending"
	PaymentStatusPaid        PaymentStatus = "paid"
	PaymentStatusFailed      PaymentStatus = "failed"
	PaymentStatusRefunded    PaymentStatus = "refunded"
	PaymentStatusNotRequired PaymentStatus = "not_required"
)

func (s PaymentStatus) Valid() bool {
	switch s {
	case PaymentStatusPending,
		PaymentStatusPaid,
		PaymentStatusFailed,
		PaymentStatusRefunded,
		PaymentStatusNotRequired:
		return true
	default:
		return false
	}
}

type OrderItem struct {
	ID        uuid.UUID
	ProductID ProductID
	Name      string
	Quantity  Quantity
	UnitPrice Money
	Subtotal  Money
	Notes     string
}

func NewOrderItem(
	productID uuid.UUID,
	name string,
	quantity int32,
	unitPrice Money,
	notes string,
) (OrderItem, error) {
	pid, err := NewProductID(productID)
	if err != nil {
		return OrderItem{}, err
	}

	qty, err := NewQuantity(quantity)
	if err != nil {
		return OrderItem{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return OrderItem{}, ErrInvalidProductName
	}

	if unitPrice.Currency == "" {
		return OrderItem{}, ErrInvalidPrice
	}

	subtotal := Money{
		MinorUnits: unitPrice.MinorUnits * int64(qty),
		Currency:   unitPrice.Currency,
	}

	return OrderItem{
		ID:        uuid.New(),
		ProductID: pid,
		Name:      name,
		Quantity:  qty,
		UnitPrice: unitPrice,
		Subtotal:  subtotal,
		Notes:     strings.TrimSpace(notes),
	}, nil
}

type Order struct {
	ID           OrderID
	TenantID     TenantID
	RestaurantID RestaurantID
	TableID      *TableID
	CustomerID   *CustomerID

	OrderNumber int64

	Type          OrderType
	State         OrderState
	PaymentStatus PaymentStatus

	Items []OrderItem

	Subtotal       Money
	TaxAmount      Money
	DiscountAmount Money
	ServiceFee     Money
	Total          Money

	Currency      string
	CustomerName  string
	CustomerPhone string
	Notes         string

	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	CancelledAt *time.Time
}

type NewOrderParams struct {
	TenantID       uuid.UUID
	RestaurantID   uuid.UUID
	TableID        *uuid.UUID
	CustomerID     *uuid.UUID
	OrderType      OrderType
	PaymentStatus  PaymentStatus
	Currency       string
	TaxAmount      int64
	DiscountAmount int64
	ServiceFee     int64
	CustomerName   string
	CustomerPhone  string
	Notes          string
	Items          []OrderItem
	Now            time.Time
}

func NewOrder(params NewOrderParams) (*Order, error) {
	tenantID, err := NewTenantID(params.TenantID)
	if err != nil {
		return nil, err
	}

	restaurantID, err := NewRestaurantID(params.RestaurantID)
	if err != nil {
		return nil, err
	}

	if !params.OrderType.Valid() {
		return nil, ErrInvalidOrderType
	}

	if !params.PaymentStatus.Valid() {
		return nil, ErrInvalidPaymentStatus
	}

	if len(params.Items) == 0 {
		return nil, ErrInvalidOrderItems
	}

	currency := strings.ToUpper(strings.TrimSpace(params.Currency))

	if len(currency) != 3 {
		return nil, ErrInvalidPrice
	}

	subtotal := Money{
		MinorUnits: 0,
		Currency:   currency,
	}

	for _, item := range params.Items {
		if item.UnitPrice.Currency != currency ||
			item.Subtotal.Currency != currency {
			return nil, ErrCurrencyMismatch
		}

		subtotal, err = subtotal.Add(item.Subtotal)
		if err != nil {
			return nil, err
		}
	}

	taxAmount, err := NewMoney(params.TaxAmount, currency)
	if err != nil {
		return nil, err
	}

	discountAmount, err := NewMoney(params.DiscountAmount, currency)
	if err != nil {
		return nil, err
	}

	serviceFee, err := NewMoney(params.ServiceFee, currency)
	if err != nil {
		return nil, err
	}

	// total = subtotal + tax + service fee - discount
	total, err := subtotal.Add(taxAmount)
	if err != nil {
		return nil, err
	}

	total, err = total.Add(serviceFee)
	if err != nil {
		return nil, err
	}

	if discountAmount.MinorUnits > total.MinorUnits {
		return nil, ErrInvalidPrice
	}

	total, err = total.Subtract(discountAmount)
	if err != nil {
		return nil, err
	}

	var tableID *TableID

	if params.TableID != nil {
		value, err := NewTableID(*params.TableID)
		if err != nil {
			return nil, err
		}

		tableID = &value
	}

	var customerID *CustomerID

	if params.CustomerID != nil {
		value, err := NewCustomerID(*params.CustomerID)
		if err != nil {
			return nil, err
		}

		customerID = &value
	}

	now := params.Now

	if now.IsZero() {
		now = time.Now().UTC()
	}

	return &Order{
		ID:             NewOrderID(),
		TenantID:       tenantID,
		RestaurantID:   restaurantID,
		TableID:        tableID,
		CustomerID:     customerID,
		Type:           params.OrderType,
		State:          OrderStatePending,
		PaymentStatus:  params.PaymentStatus,
		Items:          append([]OrderItem(nil), params.Items...),
		Subtotal:       subtotal,
		TaxAmount:      taxAmount,
		DiscountAmount: discountAmount,
		ServiceFee:     serviceFee,
		Total:          total,
		Currency:       currency,
		CustomerName:   strings.TrimSpace(params.CustomerName),
		CustomerPhone:  strings.TrimSpace(params.CustomerPhone),
		Notes:          strings.TrimSpace(params.Notes),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (o *Order) TransitionTo(next OrderState, now time.Time) error {
	if !next.Valid() {
		return ErrInvalidOrderState
	}

	if !o.State.CanTransitionTo(next) {
		return ErrInvalidStateTransition
	}

	o.State = next
	o.UpdatedAt = now

	switch next {
	case OrderStateCompleted:
		completedAt := now
		o.CompletedAt = &completedAt

	case OrderStateCancelled:
		cancelledAt := now
		o.CancelledAt = &cancelledAt
	}

	return nil
}
