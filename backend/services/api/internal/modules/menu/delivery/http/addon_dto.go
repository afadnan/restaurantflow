package httpdelivery

import (
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
)

// -----------------------------------------------------------------------------
// Add-on requests
// -----------------------------------------------------------------------------

type createAddonRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
}

type updateAddonRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
}

// -----------------------------------------------------------------------------
// Add-on responses
// -----------------------------------------------------------------------------

type addonResponse struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Price        string `json:"price"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type listAddonsResponse struct {
	Addons []addonResponse `json:"addons"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func newAddonResponse(addon *domain.Addon) addonResponse {
	return addonResponse{
		ID:           addon.ID.String(),
		TenantID:     addon.TenantID.String(),
		RestaurantID: addon.RestaurantID.String(),
		Name:         addon.Name,
		Description:  addon.Description,
		Price:        addon.Price,
		IsActive:     addon.IsActive,
		CreatedAt:    addon.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    addon.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// -----------------------------------------------------------------------------
// Item–Add-on association requests
// -----------------------------------------------------------------------------

type attachItemAddonRequest struct {
	AddonID     string `json:"addon_id"`
	IsRequired  bool   `json:"is_required"`
	MaxQuantity int32  `json:"max_quantity"`
}

type updateItemAddonRequest struct {
	IsRequired  bool  `json:"is_required"`
	MaxQuantity int32 `json:"max_quantity"`
}

// -----------------------------------------------------------------------------
// Item–Add-on association responses
// -----------------------------------------------------------------------------

type itemAddonResponse struct {
	TenantID     string `json:"tenant_id"`
	RestaurantID string `json:"restaurant_id"`
	ItemID       string `json:"item_id"`
	AddonID      string `json:"addon_id"`
	IsRequired   bool   `json:"is_required"`
	MaxQuantity  int32  `json:"max_quantity"`
}

type listItemAddonsResponse struct {
	Addons []itemAddonResponse `json:"addons"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

func newItemAddonResponse(association *domain.ItemAddon) itemAddonResponse {
	return itemAddonResponse{
		TenantID:     association.TenantID.String(),
		RestaurantID: association.RestaurantID.String(),
		ItemID:       association.ItemID.String(),
		AddonID:      association.AddonID.String(),
		IsRequired:   association.IsRequired,
		MaxQuantity:  association.MaxQuantity,
	}
}
