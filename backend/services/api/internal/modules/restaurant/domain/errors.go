package domain

import "errors"

var (
	ErrInvalidRestaurantID     = errors.New("invalid restaurant id")
	ErrInvalidTenantID         = errors.New("invalid tenant id")
	ErrInvalidRestaurantName   = errors.New("invalid restaurant name")
	ErrInvalidRestaurantSlug   = errors.New("invalid restaurant slug")
	ErrInvalidRestaurantStatus = errors.New("invalid restaurant status")

	ErrRestaurantNotFound = errors.New("restaurant not found")

	ErrRestaurantInactive  = errors.New("restaurant is inactive")
	ErrRestaurantSuspended = errors.New("restaurant is suspended")

	ErrInvalidLatitude  = errors.New("invalid latitude")
	ErrInvalidLongitude = errors.New("invalid longitude")
)
