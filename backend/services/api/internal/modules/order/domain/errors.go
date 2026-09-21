package domain

import "errors"

var (
	ErrInvalidOrderID         = errors.New("invalid order id")
	ErrInvalidTenantID        = errors.New("invalid tenant id")
	ErrInvalidRestaurantID    = errors.New("invalid restaurant id")
	ErrInvalidTableID         = errors.New("invalid table id")
	ErrInvalidCustomerID      = errors.New("invalid customer id")
	ErrInvalidOrderItems      = errors.New("order must contain at least one item")
	ErrInvalidQuantity        = errors.New("invalid quantity")
	ErrInvalidPrice           = errors.New("invalid price")
	ErrInvalidProductID       = errors.New("invalid product id")
	ErrInvalidProductName     = errors.New("invalid product name")
	ErrInvalidOrderState      = errors.New("invalid order state")
	ErrInvalidOrderType       = errors.New("invalid order type")
	ErrInvalidPaymentStatus   = errors.New("invalid payment status")
	ErrInvalidKDSState        = errors.New("invalid KDS state")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrCurrencyMismatch       = errors.New("currency mismatch")
	ErrOrderNotFound          = errors.New("order not found")
	ErrKDSNotFound            = errors.New("KDS order not found")
	ErrInsufficientStock      = errors.New("insufficient stock")
)
