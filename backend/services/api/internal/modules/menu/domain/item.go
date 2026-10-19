package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type ItemStatus string

const (
	ItemStatusActive   ItemStatus = "active"
	ItemStatusInactive ItemStatus = "inactive"
	ItemStatusSoldOut  ItemStatus = "sold_out"
)

type Item struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	RestaurantID uuid.UUID  `json:"restaurant_id"`
	CategoryID   uuid.UUID  `json:"category_id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	SKU          string     `json:"sku,omitempty"`
	Price        string     `json:"price"`
	Status       ItemStatus `json:"status"`
	ImageURL     string     `json:"image_url,omitempty"`
	SortOrder    int32      `json:"sort_order"`
	IsVegetarian bool       `json:"is_vegetarian"`
	IsVegan      bool       `json:"is_vegan"`
	IsAvailable  bool       `json:"is_available"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type CreateItemInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	CategoryID   uuid.UUID
	Name         string
	Description  string
	SKU          string
	Price        string
	Status       ItemStatus
	ImageURL     string
	SortOrder    int32
	IsVegetarian bool
	IsVegan      bool
	IsAvailable  bool
}

type UpdateItemInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	CategoryID   uuid.UUID
	Name         string
	Description  string
	SKU          string
	Price        string
	Status       ItemStatus
	ImageURL     string
	SortOrder    int32
	IsVegetarian bool
	IsVegan      bool
	IsAvailable  bool
}

func NewItem(input CreateItemInput, now time.Time) (*Item, error) {
	if input.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if input.RestaurantID == uuid.Nil {
		return nil, ErrInvalidRestaurantID
	}
	if input.CategoryID == uuid.Nil {
		return nil, ErrInvalidCategoryID
	}

	name, err := normalizeItemName(input.Name)
	if err != nil {
		return nil, err
	}
	if input.SortOrder < 0 {
		return nil, ErrInvalidSortOrder
	}
	if err := validatePrice(input.Price); err != nil {
		return nil, err
	}

	status := input.Status
	if status == "" {
		status = ItemStatusActive
	}
	if !validItemStatus(status) {
		return nil, ErrInvalidItemStatus
	}

	if input.IsVegan {
		input.IsVegetarian = true
	}

	return &Item{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		RestaurantID: input.RestaurantID,
		CategoryID:   input.CategoryID,
		Name:         name,
		Description:  strings.TrimSpace(input.Description),
		SKU:          strings.TrimSpace(input.SKU),
		Price:        input.Price,
		Status:       status,
		ImageURL:     strings.TrimSpace(input.ImageURL),
		SortOrder:    input.SortOrder,
		IsVegetarian: input.IsVegetarian,
		IsVegan:      input.IsVegan,
		IsAvailable:  input.IsAvailable,
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}, nil
}

func (i *Item) Update(input UpdateItemInput, now time.Time) error {
	if i == nil {
		return ErrInvalidItemInput
	}
	if input.TenantID == uuid.Nil || input.TenantID != i.TenantID {
		return ErrInvalidTenantID
	}
	if input.RestaurantID == uuid.Nil || input.RestaurantID != i.RestaurantID {
		return ErrInvalidRestaurantID
	}
	if input.ItemID == uuid.Nil || input.ItemID != i.ID {
		return ErrInvalidItemID
	}
	if input.CategoryID == uuid.Nil {
		return ErrInvalidCategoryID
	}

	name, err := normalizeItemName(input.Name)
	if err != nil {
		return err
	}
	if input.SortOrder < 0 {
		return ErrInvalidSortOrder
	}
	if err := validatePrice(input.Price); err != nil {
		return err
	}
	if !validItemStatus(input.Status) {
		return ErrInvalidItemStatus
	}

	i.CategoryID = input.CategoryID
	i.Name = name
	i.Description = strings.TrimSpace(input.Description)
	i.SKU = strings.TrimSpace(input.SKU)
	i.Price = input.Price
	i.Status = input.Status
	i.ImageURL = strings.TrimSpace(input.ImageURL)
	i.SortOrder = input.SortOrder
	i.IsVegetarian = input.IsVegetarian || input.IsVegan
	i.IsVegan = input.IsVegan
	i.IsAvailable = input.IsAvailable
	i.UpdatedAt = now.UTC()

	return nil
}

func normalizeItemName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 200 {
		return "", ErrInvalidItemName
	}
	return value, nil
}

func validItemStatus(status ItemStatus) bool {
	switch status {
	case ItemStatusActive, ItemStatusInactive, ItemStatusSoldOut:
		return true
	default:
		return false
	}
}

func validatePrice(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return ErrInvalidItemPrice
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return ErrInvalidItemPrice
	}
	if len(parts[0]) > 10 {
		return ErrInvalidItemPrice
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return ErrInvalidItemPrice
		}
	}
	if len(parts) == 2 {
		if len(parts[1]) > 2 || parts[1] == "" {
			return ErrInvalidItemPrice
		}
		for _, r := range parts[1] {
			if r < '0' || r > '9' {
				return ErrInvalidItemPrice
			}
		}
	}
	return nil
}
