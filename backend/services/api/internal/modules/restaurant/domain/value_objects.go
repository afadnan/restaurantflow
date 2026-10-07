package domain

import (
	"strings"

	"github.com/google/uuid"
)

type RestaurantID struct {
	value uuid.UUID
}

func NewRestaurantID() RestaurantID {
	return RestaurantID{
		value: uuid.New(),
	}
}

func NewRestaurantIDFromUUID(value uuid.UUID) (RestaurantID, error) {
	if value == uuid.Nil {
		return RestaurantID{}, ErrInvalidRestaurantID
	}

	return RestaurantID{
		value: value,
	}, nil
}

func (id RestaurantID) UUID() uuid.UUID {
	return id.value
}

type TenantID struct {
	value uuid.UUID
}

func NewTenantIDFromUUID(value uuid.UUID) (TenantID, error) {
	if value == uuid.Nil {
		return TenantID{}, ErrInvalidTenantID
	}

	return TenantID{
		value: value,
	}, nil
}

func (id TenantID) UUID() uuid.UUID {
	return id.value
}

type RestaurantName string

func NewRestaurantName(value string) (RestaurantName, error) {
	value = strings.TrimSpace(value)

	if value == "" || len(value) > 200 {
		return "", ErrInvalidRestaurantName
	}

	return RestaurantName(value), nil
}

func (name RestaurantName) String() string {
	return string(name)
}

type RestaurantSlug string

func NewRestaurantSlug(value string) (RestaurantSlug, error) {
	value = strings.ToLower(strings.TrimSpace(value))

	if value == "" || len(value) > 200 {
		return "", ErrInvalidRestaurantSlug
	}

	return RestaurantSlug(value), nil
}

func (slug RestaurantSlug) String() string {
	return string(slug)
}

type RestaurantStatus string

const (
	RestaurantStatusActive    RestaurantStatus = "active"
	RestaurantStatusInactive  RestaurantStatus = "inactive"
	RestaurantStatusSuspended RestaurantStatus = "suspended"
)

func (status RestaurantStatus) Valid() bool {
	switch status {
	case RestaurantStatusActive,
		RestaurantStatusInactive,
		RestaurantStatusSuspended:
		return true

	default:
		return false
	}
}
