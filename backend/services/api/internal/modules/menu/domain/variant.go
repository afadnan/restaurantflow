package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidVariantID    = errors.New("invalid menu variant id")
	ErrInvalidVariantName  = errors.New("menu variant name must be between 1 and 200 characters")
	ErrInvalidVariantInput = errors.New("invalid menu variant input")
	ErrInvalidPriceDelta   = errors.New("menu variant price delta must be a decimal with at most 10 integer digits and 2 fractional digits")
	ErrVariantNotFound     = errors.New("menu variant not found")
	ErrVariantInactive     = errors.New("menu variant is inactive")
)

var priceDeltaPattern = regexp.MustCompile(`^-?[0-9]{1,10}(\.[0-9]{1,2})?$`)

type Variant struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	ItemID       uuid.UUID `json:"item_id"`
	Name         string    `json:"name"`
	PriceDelta   string    `json:"price_delta"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewVariant(
	tenantID, restaurantID, itemID uuid.UUID,
	name, priceDelta string,
	now time.Time,
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

	priceDelta = strings.TrimSpace(priceDelta)
	if err := validatePriceDelta(priceDelta); err != nil {
		return nil, err
	}

	now = now.UTC()

	return &Variant{
		ID:           uuid.New(),
		TenantID:     tenantID,
		RestaurantID: restaurantID,
		ItemID:       itemID,
		Name:         name,
		PriceDelta:   priceDelta,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (v *Variant) Update(
	tenantID, restaurantID, itemID, variantID uuid.UUID,
	name, priceDelta string,
	now time.Time,
) error {
	if v == nil {
		return ErrInvalidVariantInput
	}
	if tenantID == uuid.Nil || tenantID != v.TenantID {
		return ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil || restaurantID != v.RestaurantID {
		return ErrInvalidRestaurantID
	}
	if itemID == uuid.Nil || itemID != v.ItemID {
		return ErrInvalidItemID
	}
	if variantID == uuid.Nil || variantID != v.ID {
		return ErrInvalidVariantID
	}
	if !v.IsActive {
		return ErrVariantInactive
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return ErrInvalidVariantName
	}

	priceDelta = strings.TrimSpace(priceDelta)
	if err := validatePriceDelta(priceDelta); err != nil {
		return err
	}

	v.Name = name
	v.PriceDelta = priceDelta
	v.UpdatedAt = now.UTC()

	return nil
}

func (v *Variant) Deactivate(
	tenantID, restaurantID, itemID, variantID uuid.UUID,
	now time.Time,
) error {
	if v == nil {
		return ErrInvalidVariantInput
	}
	if tenantID == uuid.Nil || tenantID != v.TenantID {
		return ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil || restaurantID != v.RestaurantID {
		return ErrInvalidRestaurantID
	}
	if itemID == uuid.Nil || itemID != v.ItemID {
		return ErrInvalidItemID
	}
	if variantID == uuid.Nil || variantID != v.ID {
		return ErrInvalidVariantID
	}
	if !v.IsActive {
		return ErrVariantInactive
	}

	v.IsActive = false
	v.UpdatedAt = now.UTC()

	return nil
}

func validatePriceDelta(value string) error {
	if !priceDeltaPattern.MatchString(value) {
		return ErrInvalidPriceDelta
	}
	return nil
}
