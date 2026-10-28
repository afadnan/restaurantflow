package httpdelivery

import (
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
)

type createItemRequest struct {
	CategoryID   string            `json:"category_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	SKU          string            `json:"sku"`
	Price        string            `json:"price"`
	Status       domain.ItemStatus `json:"status"`
	ImageURL     string            `json:"image_url"`
	SortOrder    int32             `json:"sort_order"`
	IsVegetarian bool              `json:"is_vegetarian"`
	IsVegan      bool              `json:"is_vegan"`
	IsAvailable  bool              `json:"is_available"`
}

type updateItemRequest = createItemRequest

type itemResponse struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	RestaurantID string            `json:"restaurant_id"`
	CategoryID   string            `json:"category_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	SKU          string            `json:"sku"`
	Price        string            `json:"price"`
	Status       domain.ItemStatus `json:"status"`
	ImageURL     string            `json:"image_url"`
	SortOrder    int32             `json:"sort_order"`
	IsVegetarian bool              `json:"is_vegetarian"`
	IsVegan      bool              `json:"is_vegan"`
	IsAvailable  bool              `json:"is_available"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

type listItemsResponse struct {
	Items  []itemResponse `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func newItemResponse(item *domain.Item) itemResponse {
	return itemResponse{
		ID:           item.ID.String(),
		TenantID:     item.TenantID.String(),
		RestaurantID: item.RestaurantID.String(),
		CategoryID:   item.CategoryID.String(),
		Name:         item.Name,
		Description:  item.Description,
		SKU:          item.SKU,
		Price:        item.Price,
		Status:       item.Status,
		ImageURL:     item.ImageURL,
		SortOrder:    item.SortOrder,
		IsVegetarian: item.IsVegetarian,
		IsVegan:      item.IsVegan,
		IsAvailable:  item.IsAvailable,
		CreatedAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
