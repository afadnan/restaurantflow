package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Restaurant struct {
	ID           RestaurantID
	TenantID     TenantID
	Name         RestaurantName
	Slug         RestaurantSlug
	Description  string
	Phone        string
	Email        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Country      string
	Latitude     *float64
	Longitude    *float64
	Status       RestaurantStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateRestaurantInput struct {
	TenantID     uuid.UUID
	Name         string
	Slug         string
	Description  string
	Phone        string
	Email        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Country      string
	Latitude     *float64
	Longitude    *float64
}

type UpdateRestaurantInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	Name         string
	Slug         string
	Description  string
	Phone        string
	Email        string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Country      string
	Latitude     *float64
	Longitude    *float64
}

func NewRestaurant(
	input CreateRestaurantInput,
	now time.Time,
) (*Restaurant, error) {
	if input.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	name, err := NewRestaurantName(input.Name)
	if err != nil {
		return nil, err
	}

	slug, err := NewRestaurantSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	country := strings.TrimSpace(input.Country)
	if country == "" {
		country = "India"
	}

	if err := validateCoordinates(input.Latitude, input.Longitude); err != nil {
		return nil, err
	}

	tenantID, err := NewTenantIDFromUUID(input.TenantID)
	if err != nil {
		return nil, err
	}

	return &Restaurant{
		ID:           NewRestaurantID(),
		TenantID:     tenantID,
		Name:         name,
		Slug:         slug,
		Description:  strings.TrimSpace(input.Description),
		Phone:        strings.TrimSpace(input.Phone),
		Email:        strings.TrimSpace(input.Email),
		AddressLine1: strings.TrimSpace(input.AddressLine1),
		AddressLine2: strings.TrimSpace(input.AddressLine2),
		City:         strings.TrimSpace(input.City),
		State:        strings.TrimSpace(input.State),
		PostalCode:   strings.TrimSpace(input.PostalCode),
		Country:      country,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		Status:       RestaurantStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (r *Restaurant) Update(
	input UpdateRestaurantInput,
	now time.Time,
) error {
	if r == nil {
		return errors.New("restaurant is required")
	}

	if input.TenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	if input.TenantID != r.TenantID.UUID() {
		return ErrInvalidTenantID
	}

	if input.RestaurantID == uuid.Nil ||
		input.RestaurantID != r.ID.UUID() {
		return ErrInvalidRestaurantID
	}

	name, err := NewRestaurantName(input.Name)
	if err != nil {
		return err
	}

	slug, err := NewRestaurantSlug(input.Slug)
	if err != nil {
		return err
	}

	country := strings.TrimSpace(input.Country)
	if country == "" {
		country = "India"
	}

	if err := validateCoordinates(input.Latitude, input.Longitude); err != nil {
		return err
	}

	r.Name = name
	r.Slug = slug
	r.Description = strings.TrimSpace(input.Description)
	r.Phone = strings.TrimSpace(input.Phone)
	r.Email = strings.TrimSpace(input.Email)
	r.AddressLine1 = strings.TrimSpace(input.AddressLine1)
	r.AddressLine2 = strings.TrimSpace(input.AddressLine2)
	r.City = strings.TrimSpace(input.City)
	r.State = strings.TrimSpace(input.State)
	r.PostalCode = strings.TrimSpace(input.PostalCode)
	r.Country = country
	r.Latitude = input.Latitude
	r.Longitude = input.Longitude
	r.UpdatedAt = now

	return nil
}

func (r *Restaurant) Activate(now time.Time) error {
	if r == nil {
		return errors.New("restaurant is required")
	}

	if r.Status == RestaurantStatusActive {
		return nil
	}

	r.Status = RestaurantStatusActive
	r.UpdatedAt = now

	return nil
}

func (r *Restaurant) Deactivate(now time.Time) error {
	if r == nil {
		return errors.New("restaurant is required")
	}

	if r.Status == RestaurantStatusInactive {
		return nil
	}

	r.Status = RestaurantStatusInactive
	r.UpdatedAt = now

	return nil
}

func (r *Restaurant) Suspend(now time.Time) error {
	if r == nil {
		return errors.New("restaurant is required")
	}

	if r.Status == RestaurantStatusSuspended {
		return nil
	}

	r.Status = RestaurantStatusSuspended
	r.UpdatedAt = now

	return nil
}

func (r *Restaurant) CanAcceptOrders() error {
	if r == nil {
		return errors.New("restaurant is required")
	}

	switch r.Status {
	case RestaurantStatusActive:
		return nil

	case RestaurantStatusInactive:
		return ErrRestaurantInactive

	case RestaurantStatusSuspended:
		return ErrRestaurantSuspended

	default:
		return ErrInvalidRestaurantStatus
	}
}

func validateCoordinates(
	latitude *float64,
	longitude *float64,
) error {
	if latitude != nil {
		if *latitude < -90 || *latitude > 90 {
			return ErrInvalidLatitude
		}
	}

	if longitude != nil {
		if *longitude < -180 || *longitude > 180 {
			return ErrInvalidLongitude
		}
	}

	return nil
}
