package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

type GetRestaurantUseCase struct {
	DB          *postgres.DB
	Restaurants domain.RestaurantRepository
}

type GetRestaurantInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
}

func (uc *GetRestaurantUseCase) Execute(
	ctx context.Context,
	input GetRestaurantInput,
) (*domain.Restaurant, error) {
	if uc == nil || uc.DB == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if uc.Restaurants == nil {
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

	return restaurant, nil
}
