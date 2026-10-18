package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SortOrder    int32     `json:"sort_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateCategoryInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	Name         string
	Description  string
	SortOrder    int32
}

type UpdateCategoryInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	CategoryID   uuid.UUID
	Name         string
	Description  string
	SortOrder    int32
}

func NewCategory(input CreateCategoryInput, now time.Time) (*Category, error) {
	if input.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	if input.RestaurantID == uuid.Nil {
		return nil, ErrInvalidRestaurantID
	}

	name, err := normalizeCategoryName(input.Name)
	if err != nil {
		return nil, err
	}

	if input.SortOrder < 0 {
		return nil, ErrInvalidSortOrder
	}

	return &Category{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		RestaurantID: input.RestaurantID,
		Name:         name,
		Description:  strings.TrimSpace(input.Description),
		SortOrder:    input.SortOrder,
		IsActive:     true,
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}, nil
}

func (c *Category) Update(input UpdateCategoryInput, now time.Time) error {
	if c == nil {
		return ErrInvalidCategoryInput
	}

	if input.TenantID == uuid.Nil || input.TenantID != c.TenantID {
		return ErrInvalidTenantID
	}

	if input.RestaurantID == uuid.Nil || input.RestaurantID != c.RestaurantID {
		return ErrInvalidRestaurantID
	}

	if input.CategoryID == uuid.Nil || input.CategoryID != c.ID {
		return ErrInvalidCategoryID
	}

	name, err := normalizeCategoryName(input.Name)
	if err != nil {
		return err
	}

	if input.SortOrder < 0 {
		return ErrInvalidSortOrder
	}

	c.Name = name
	c.Description = strings.TrimSpace(input.Description)
	c.SortOrder = input.SortOrder
	c.UpdatedAt = now.UTC()

	return nil
}

func normalizeCategoryName(value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" || len(value) > 200 {
		return "", ErrInvalidCategoryName
	}

	return value, nil
}
