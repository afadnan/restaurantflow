package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type DeactivateRestaurantUseCase struct {
	DB          *postgres.DB
	Restaurants domain.RestaurantRepository
	Events      sharedevents.EventRepository
	Clock       Clock
}

type DeactivateRestaurantInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
}

func (uc *DeactivateRestaurantUseCase) Execute(
	ctx context.Context,
	input DeactivateRestaurantInput,
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

	if err := restaurant.Deactivate(now); err != nil {
		return nil, err
	}

	restaurant, err = uc.Restaurants.Deactivate(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
	)
	if err != nil {
		return nil, err
	}

	event := domain.RestaurantDeactivatedEvent{
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

func (uc *DeactivateRestaurantUseCase) now() time.Time {
	if uc.Clock != nil {
		return uc.Clock.Now()
	}

	return time.Now().UTC()
}
