package domain

import "errors"

var (
	ErrInvalidOrderID         = errors.New("invalid order id")
	ErrInvalidTenantID        = errors.New("invalid tenant id")
	ErrInvalidCustomerID      = errors.New("invalid customer id")
	ErrInvalidOrderItems      = errors.New("order must contain at least one item")
	ErrInvalidQuantity        = errors.New("quantity must be greater than zero")
	ErrInvalidPrice           = errors.New("price must be greater than or equal to zero")
	ErrInvalidProductID       = errors.New("invalid product id")
	ErrInvalidOrderState      = errors.New("invalid order state")
	ErrInvalidKDSState        = errors.New("invalid kds state")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrOrderNotFound          = errors.New("order not found")
	ErrKDSNotFound            = errors.New("kds order not found")
	ErrInsufficientStock      = errors.New("insufficient stock")
)
