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

	ErrItemNotFound = errors.New("menu item not found")

	// addon errors
	ErrInvalidAddonID          = errors.New("invalid menu addon id")
	ErrInvalidAddonName        = errors.New("menu addon name must be between 1 and 200 characters")
	ErrInvalidAddonPrice       = errors.New("menu addon price must be a non-negative decimal with at most two decimal places")
	ErrInvalidAddonInput       = errors.New("invalid menu addon input")
	ErrAddonNotFound           = errors.New("menu addon not found")
	ErrAddonInactive           = errors.New("menu addon is inactive")
	ErrInvalidAddonMaxQuantity = errors.New("menu addon max quantity must be greater than zero")
	ErrInvalidItemAddonInput   = errors.New("invalid menu item addon input")
	ErrItemAddonNotFound       = errors.New("menu item addon association not found")
	ErrItemAddonAlreadyExists  = errors.New("menu item addon association already exists")
)
