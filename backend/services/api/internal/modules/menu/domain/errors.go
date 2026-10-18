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
)
