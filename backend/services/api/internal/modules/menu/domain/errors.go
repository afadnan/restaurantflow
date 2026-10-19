package domain

import "errors"

var (
	ErrInvalidTenantID      = errors.New("invalid tenant id")
	ErrInvalidRestaurantID  = errors.New("invalid restaurant id")
	ErrInvalidCategoryID    = errors.New("invalid category id")
	ErrInvalidCategoryName  = errors.New("category name must be between 1 and 200 characters")
	ErrInvalidSortOrder     = errors.New("category sort order cannot be negative")
	ErrCategoryNotFound     = errors.New("menu category not found")
	ErrInvalidCategoryInput = errors.New("invalid menu category input")
	ErrInvalidItemID        = errors.New("invalid menu item id")
	ErrInvalidItemName      = errors.New("menu item name must be between 1 and 200 characters")
	ErrInvalidItemStatus    = errors.New("invalid menu item status")
	ErrInvalidItemInput     = errors.New("invalid menu item input")
	ErrInvalidItemPrice     = errors.New("menu item price must be a non-negative decimal with at most two decimal places")
)
