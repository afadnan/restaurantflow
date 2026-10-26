package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Addon struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        string    `json:"price"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewAddon(
	tenantID, restaurantID uuid.UUID,
	name, description, price string,
	now time.Time,
) (*Addon, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil {
		return nil, ErrInvalidRestaurantID
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return nil, ErrInvalidAddonName
	}

	price = strings.TrimSpace(price)
	if err := validatePrice(price); err != nil {
		return nil, ErrInvalidAddonPrice
	}

	now = now.UTC()

	return &Addon{
		ID:           uuid.New(),
		TenantID:     tenantID,
		RestaurantID: restaurantID,
		Name:         name,
		Description:  strings.TrimSpace(description),
		Price:        price,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (a *Addon) Update(
	tenantID, restaurantID, addonID uuid.UUID,
	name, description, price string,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidAddonInput
	}
	if tenantID == uuid.Nil || tenantID != a.TenantID {
		return ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil || restaurantID != a.RestaurantID {
		return ErrInvalidRestaurantID
	}
	if addonID == uuid.Nil || addonID != a.ID {
		return ErrInvalidAddonID
	}
	if !a.IsActive {
		return ErrAddonInactive
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return ErrInvalidAddonName
	}

	price = strings.TrimSpace(price)
	if err := validatePrice(price); err != nil {
		return ErrInvalidAddonPrice
	}

	a.Name = name
	a.Description = strings.TrimSpace(description)
	a.Price = price
	a.UpdatedAt = now.UTC()

	return nil
}

func (a *Addon) Deactivate(
	tenantID, restaurantID, addonID uuid.UUID,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidAddonInput
	}
	if tenantID == uuid.Nil || tenantID != a.TenantID {
		return ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil || restaurantID != a.RestaurantID {
		return ErrInvalidRestaurantID
	}
	if addonID == uuid.Nil || addonID != a.ID {
		return ErrInvalidAddonID
	}
	if !a.IsActive {
		return ErrAddonInactive
	}

	a.IsActive = false
	a.UpdatedAt = now.UTC()

	return nil
}
