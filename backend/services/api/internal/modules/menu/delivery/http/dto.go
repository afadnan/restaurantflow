package httpdelivery

import (
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
)

type createCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int32  `json:"sort_order"`
}

type updateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int32  `json:"sort_order"`
}

type categoryResponse struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	SortOrder    int32  `json:"sort_order"`
	IsActive     bool   `json:"is_active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type listCategoriesResponse struct {
	Categories []categoryResponse `json:"categories"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

func newCategoryResponse(category *domain.Category) categoryResponse {
	return categoryResponse{
		ID:           category.ID.String(),
		TenantID:     category.TenantID.String(),
		RestaurantID: category.RestaurantID.String(),
		Name:         category.Name,
		Description:  category.Description,
		SortOrder:    category.SortOrder,
		IsActive:     category.IsActive,
		CreatedAt:    category.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    category.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
