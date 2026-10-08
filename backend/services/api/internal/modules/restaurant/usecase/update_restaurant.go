package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type UpdateRestaurantUseCase struct {
	DB          *postgres.DB
	Restaurants domain.RestaurantRepository
	Events      sharedevents.EventRepository
	Clock       Clock
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

func (uc *UpdateRestaurantUseCase) Execute(
	ctx context.Context,
	input UpdateRestaurantInput,
) (*domain.Restaurant, error) {
	if uc == nil || uc.DB == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if uc.Restaurants == nil || uc.Events == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if input.RestaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	restaurant, err := uc.Restaurants.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
	)
	if err != nil {
		return nil, err
	}

	now := uc.now()

	if err := restaurant.Update(
		domain.UpdateRestaurantInput{
			TenantID:     input.TenantID,
			RestaurantID: input.RestaurantID,
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
	); err != nil {
		return nil, err
	}

	if err := uc.Restaurants.Update(ctx, tx, restaurant); err != nil {
		return nil, err
	}

	event := domain.RestaurantUpdatedEvent{
		ID:         uuid.New(),
		Tenant:     restaurant.TenantID.UUID(),
		Restaurant: restaurant.ID.UUID(),
		Occurred:   restaurant.UpdatedAt,
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return restaurant, nil
}

func (uc *UpdateRestaurantUseCase) now() time.Time {
	if uc.Clock != nil {
		return uc.Clock.Now()
	}

	return time.Now().UTC()
}
