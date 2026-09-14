package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type TenantID uuid.UUID

func NewTenantID(id uuid.UUID) (TenantID, error) {
	if id == uuid.Nil {
		return TenantID{}, ErrInvalidTenantID
	}

	return TenantID(id), nil
}

func (id TenantID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

type OrderID uuid.UUID

func NewOrderID() OrderID {
	return OrderID(uuid.New())
}

func ParseOrderID(value string) (OrderID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return OrderID{}, ErrInvalidOrderID
	}

	return OrderID(id), nil
}

func (id OrderID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

type CustomerID uuid.UUID

func NewCustomerID(id uuid.UUID) (CustomerID, error) {
	if id == uuid.Nil {
		return CustomerID{}, ErrInvalidCustomerID
	}

	return CustomerID(id), nil
}

func (id CustomerID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

type ProductID uuid.UUID

func NewProductID(id uuid.UUID) (ProductID, error) {
	if id == uuid.Nil {
		return ProductID{}, ErrInvalidProductID
	}

	return ProductID(id), nil
}

func (id ProductID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

type Money struct {
	MinorUnits int64
	Currency   string
}

func NewMoney(minorUnits int64, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if minorUnits < 0 {
		return Money{}, ErrInvalidPrice
	}

	if len(currency) != 3 {
		return Money{}, fmt.Errorf("invalid currency: %s", currency)
	}

	return Money{
		MinorUnits: minorUnits,
		Currency:   currency,
	}, nil
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s/%s", m.Currency, other.Currency)
	}

	return Money{
		MinorUnits: m.MinorUnits + other.MinorUnits,
		Currency:   m.Currency,
	}, nil
}

type Quantity int32

func NewQuantity(value int32) (Quantity, error) {
	if value <= 0 {
		return 0, ErrInvalidQuantity
	}

	return Quantity(value), nil
}

func (q Quantity) Int32() int32 {
	return int32(q)
}
