package domain

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

type TenantID struct {
	value uuid.UUID
}

func marshalUUID(id uuid.UUID) ([]byte, error) {
	return json.Marshal(id.String())
}

func (id TenantID) MarshalJSON() ([]byte, error) {
	return marshalUUID(id.value)
}

func NewTenantID(id uuid.UUID) (TenantID, error) {
	if id == uuid.Nil {
		return TenantID{}, ErrInvalidTenantID
	}

	return TenantID{value: id}, nil
}

func (id TenantID) UUID() uuid.UUID {
	return id.value
}

type RestaurantID struct {
	value uuid.UUID
}

func (id RestaurantID) MarshalJSON() ([]byte, error) {
	return marshalUUID(id.value)
}

func NewRestaurantID(id uuid.UUID) (RestaurantID, error) {
	if id == uuid.Nil {
		return RestaurantID{}, ErrInvalidRestaurantID
	}

	return RestaurantID{value: id}, nil
}

func (id RestaurantID) UUID() uuid.UUID {
	return id.value
}

type TableID struct {
	value uuid.UUID
}

func (id TableID) MarshalJSON() ([]byte, error) {
	return marshalUUID(id.value)
}

func NewTableID(id uuid.UUID) (TableID, error) {
	if id == uuid.Nil {
		return TableID{}, ErrInvalidTableID
	}

	return TableID{value: id}, nil
}

func (id TableID) UUID() uuid.UUID {
	return id.value
}

type OrderID struct {
	value uuid.UUID
}

func (id OrderID) MarshalJSON() ([]byte, error) {
	return marshalUUID(id.value)
}

func NewOrderID() OrderID {
	return OrderID{
		value: uuid.New(),
	}
}

func NewOrderIDFromUUID(id uuid.UUID) (OrderID, error) {
	if id == uuid.Nil {
		return OrderID{}, ErrInvalidOrderID
	}

	return OrderID{value: id}, nil
}

func (id OrderID) UUID() uuid.UUID {
	return id.value
}

type CustomerID struct {
	value uuid.UUID
}

func (id CustomerID) MarshalJSON() ([]byte, error) {
	return marshalUUID(id.value)
}

func NewCustomerID(id uuid.UUID) (CustomerID, error) {
	if id == uuid.Nil {
		return CustomerID{}, ErrInvalidCustomerID
	}

	return CustomerID{value: id}, nil
}

func (id CustomerID) UUID() uuid.UUID {
	return id.value
}

type ProductID struct {
	value uuid.UUID
}

func (id ProductID) MarshalJSON() ([]byte, error) {
	return marshalUUID(id.value)
}

func NewProductID(id uuid.UUID) (ProductID, error) {
	if id == uuid.Nil {
		return ProductID{}, ErrInvalidProductID
	}

	return ProductID{value: id}, nil
}

func (id ProductID) UUID() uuid.UUID {
	return id.value
}

type Money struct {
	MinorUnits int64
	Currency   string
}

func NewMoney(minorUnits int64, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if len(currency) != 3 {
		return Money{}, ErrInvalidPrice
	}

	if minorUnits < 0 {
		return Money{}, ErrInvalidPrice
	}

	return Money{
		MinorUnits: minorUnits,
		Currency:   currency,
	}, nil
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, ErrCurrencyMismatch
	}

	return Money{
		MinorUnits: m.MinorUnits + other.MinorUnits,
		Currency:   m.Currency,
	}, nil
}

func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, ErrCurrencyMismatch
	}

	if other.MinorUnits > m.MinorUnits {
		return Money{}, ErrInvalidPrice
	}

	return Money{
		MinorUnits: m.MinorUnits - other.MinorUnits,
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

func (q Quantity) Value() int32 {
	return int32(q)
}
