package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type CreateRestaurantUseCase struct {
	DB          *postgres.DB
	Restaurants domain.RestaurantRepository
	Events      sharedevents.EventRepository
	Clock       Clock
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

func (uc *CreateRestaurantUseCase) Execute(
	ctx context.Context,
	input CreateRestaurantInput,
) (*domain.Restaurant, error) {
	if uc == nil || uc.DB == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if uc.Restaurants == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if uc.Events == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	now := uc.now()

	restaurant, err := domain.NewRestaurant(
		domain.CreateRestaurantInput{
			TenantID:     input.TenantID,
			Name:         input.Name,
			Slug:         input.Slug,
			Description:  input.Description,
			Phone:        input.Phone,
			Email:        input.Email,
			AddressLine1: input.AddressLine1,
			AddressLine2: input.AddressLine2,
			City:         input.City,
			State:        input.State,
			PostalCode:   input.PostalCode,
			Country:      input.Country,
			Latitude:     input.Latitude,
			Longitude:    input.Longitude,
		},
		now,
	)
	if err != nil {
		return nil, err
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := uc.Restaurants.Create(ctx, tx, restaurant); err != nil {
		return nil, err
	}

	event := domain.RestaurantCreatedEvent{
		ID:         uuid.New(),
		Tenant:     restaurant.TenantID.UUID(),
		Restaurant: restaurant.ID.UUID(),
		Occurred:   restaurant.CreatedAt,
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return restaurant, nil
}

func (uc *CreateRestaurantUseCase) now() time.Time {
	if uc.Clock != nil {
		return uc.Clock.Now()
	}

	return time.Now().UTC()
}
