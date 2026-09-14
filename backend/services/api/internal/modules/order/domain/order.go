package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderState string

const (
	OrderStatePending   OrderState = "pending"
	OrderStatePreparing OrderState = "preparing"
	OrderStateReady     OrderState = "ready"
	OrderStateCompleted OrderState = "completed"
)

func (s OrderState) Valid() bool {
	switch s {
	case OrderStatePending,
		OrderStatePreparing,
		OrderStateReady,
		OrderStateCompleted:
		return true
	default:
		return false
	}
}

func (s OrderState) CanTransitionTo(next OrderState) bool {
	switch s {
	case OrderStatePending:
		return next == OrderStatePreparing

	case OrderStatePreparing:
		return next == OrderStateReady

	case OrderStateReady:
		return next == OrderStateCompleted

	case OrderStateCompleted:
		return false

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
}

func NewOrderItem(
	productID uuid.UUID,
	name string,
	quantity int32,
	unitPrice Money,
) (OrderItem, error) {
	pid, err := NewProductID(productID)
	if err != nil {
		return OrderItem{}, err
	}

	qty, err := NewQuantity(quantity)
	if err != nil {
		return OrderItem{}, err
	}

	if name == "" {
		return OrderItem{}, ErrInvalidProductID
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
	}, nil
}

type Order struct {
	ID         OrderID
	TenantID   TenantID
	CustomerID CustomerID
	State      OrderState
	Items      []OrderItem
	Total      Money
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewOrder(
	tenantID uuid.UUID,
	customerID uuid.UUID,
	items []OrderItem,
	now time.Time,
) (*Order, error) {
	tid, err := NewTenantID(tenantID)
	if err != nil {
		return nil, err
	}

	cid, err := NewCustomerID(customerID)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, ErrInvalidOrderItems
	}

	var total Money

	for index, item := range items {
		if index == 0 {
			total = item.Subtotal
			continue
		}

		total, err = total.Add(item.Subtotal)
		if err != nil {
			return nil, err
		}
	}

	return &Order{
		ID:         NewOrderID(),
		TenantID:   tid,
		CustomerID: cid,
		State:      OrderStatePending,
		Items:      append([]OrderItem(nil), items...),
		Total:      total,
		CreatedAt:  now,
		UpdatedAt:  now,
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

	return nil
}
