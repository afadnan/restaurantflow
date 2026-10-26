package domain

import "github.com/google/uuid"

type ItemAddon struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	ItemID       uuid.UUID `json:"menu_item_id"`
	AddonID      uuid.UUID `json:"addon_id"`
	IsRequired   bool      `json:"is_required"`
	MaxQuantity  int32     `json:"max_quantity"`
}

func NewItemAddon(
	tenantID, restaurantID, itemID, addonID uuid.UUID,
	isRequired bool,
	maxQuantity int32,
) (*ItemAddon, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil {
		return nil, ErrInvalidRestaurantID
	}
	if itemID == uuid.Nil {
		return nil, ErrInvalidItemID
	}
	if addonID == uuid.Nil {
		return nil, ErrInvalidAddonID
	}
	if maxQuantity <= 0 {
		return nil, ErrInvalidAddonMaxQuantity
	}

	return &ItemAddon{
		TenantID:     tenantID,
		RestaurantID: restaurantID,
		ItemID:       itemID,
		AddonID:      addonID,
		IsRequired:   isRequired,
		MaxQuantity:  maxQuantity,
	}, nil
}

func (ia *ItemAddon) Update(
	tenantID, restaurantID, itemID, addonID uuid.UUID,
	isRequired bool,
	maxQuantity int32,
) error {
	if ia == nil {
		return ErrInvalidItemAddonInput
	}
	if tenantID == uuid.Nil || tenantID != ia.TenantID {
		return ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil || restaurantID != ia.RestaurantID {
		return ErrInvalidRestaurantID
	}
	if itemID == uuid.Nil || itemID != ia.ItemID {
		return ErrInvalidItemID
	}
	if addonID == uuid.Nil || addonID != ia.AddonID {
		return ErrInvalidAddonID
	}
	if maxQuantity <= 0 {
		return ErrInvalidAddonMaxQuantity
	}

	ia.IsRequired = isRequired
	ia.MaxQuantity = maxQuantity

	return nil
}
