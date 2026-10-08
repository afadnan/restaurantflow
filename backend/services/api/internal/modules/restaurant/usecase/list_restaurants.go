package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

const (
	DefaultRestaurantListLimit = 50
	MaxRestaurantListLimit     = 100
)

type ListRestaurantsUseCase struct {
	DB          *postgres.DB
	Restaurants domain.RestaurantRepository
}

type ListRestaurantsInput struct {
	TenantID uuid.UUID
	Limit    int
	Offset   int
}

type ListRestaurantsOutput struct {
	Restaurants []*domain.Restaurant
	Limit       int
	Offset      int
}

func (uc *ListRestaurantsUseCase) Execute(
	ctx context.Context,
	input ListRestaurantsInput,
) (*ListRestaurantsOutput, error) {
	if uc == nil || uc.DB == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if uc.Restaurants == nil {
		return nil, ErrUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	limit := input.Limit
	if limit == 0 {
		limit = DefaultRestaurantListLimit
	}

	if limit < 1 || limit > MaxRestaurantListLimit {
		return nil, ErrInvalidPagination
	}

	if input.Offset < 0 {
		return nil, ErrInvalidPagination
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	restaurants, err := uc.Restaurants.List(
		ctx,
		tx,
		input.TenantID,
		limit,
		input.Offset,
	)
	if err != nil {
		return nil, err
	}

	return &ListRestaurantsOutput{
		Restaurants: restaurants,
		Limit:       limit,
		Offset:      input.Offset,
	}, nil
}
