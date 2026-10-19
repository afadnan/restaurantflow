package domain

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidVariantID    = errors.New("invalid menu variant id")
	ErrInvalidVariantName  = errors.New("menu variant name must be between 1 and 200 characters")
	ErrInvalidVariantInput = errors.New("invalid menu variant input")
)

type Variant struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	ItemID       uuid.UUID `json:"item_id"`
	Name         string    `json:"name"`
	PriceDelta   string    `json:"price_delta"`
	IsActive     bool      `json:"is_active"`
}

func NewVariant(
	tenantID, restaurantID, itemID uuid.UUID,
	name, priceDelta string,
) (*Variant, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil {
		return nil, ErrInvalidRestaurantID
	}
	if itemID == uuid.Nil {
		return nil, ErrInvalidItemID
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return nil, ErrInvalidVariantName
	}
	if err := validatePrice(priceDelta); err != nil {
		return nil, err
	}

	return &Variant{
		ID:           uuid.New(),
		TenantID:     tenantID,
		RestaurantID: restaurantID,
		ItemID:       itemID,
		Name:         name,
		PriceDelta:   priceDelta,
		IsActive:     true,
	}, nil
}
