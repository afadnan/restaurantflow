package httpdelivery

import (
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
)

type createRestaurantRequest struct {
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	AddressLine1 string   `json:"address_line1"`
	AddressLine2 string   `json:"address_line2"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	PostalCode   string   `json:"postal_code"`
	Country      string   `json:"country"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
}

type updateRestaurantRequest struct {
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	AddressLine1 string   `json:"address_line1"`
	AddressLine2 string   `json:"address_line2"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	PostalCode   string   `json:"postal_code"`
	Country      string   `json:"country"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
}

type restaurantResponse struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	AddressLine1 string   `json:"address_line1"`
	AddressLine2 string   `json:"address_line2"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	PostalCode   string   `json:"postal_code"`
	Country      string   `json:"country"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type listRestaurantsResponse struct {
	Restaurants []restaurantResponse `json:"restaurants"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
}

func newRestaurantResponse(
	restaurant *domain.Restaurant,
) restaurantResponse {
	return restaurantResponse{
		ID:           restaurant.ID.UUID().String(),
		TenantID:     restaurant.TenantID.UUID().String(),
		Name:         restaurant.Name.String(),
		Slug:         restaurant.Slug.String(),
		Description:  restaurant.Description,
		Phone:        restaurant.Phone,
		Email:        restaurant.Email,
		AddressLine1: restaurant.AddressLine1,
		AddressLine2: restaurant.AddressLine2,
		City:         restaurant.City,
		State:        restaurant.State,
		PostalCode:   restaurant.PostalCode,
		Country:      restaurant.Country,
		Latitude:     restaurant.Latitude,
		Longitude:    restaurant.Longitude,
		Status:       string(restaurant.Status),
		CreatedAt:    restaurant.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    restaurant.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
