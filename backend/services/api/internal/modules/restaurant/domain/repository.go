package domain

import (
	"context"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
)

type Transaction = sharedevents.Transaction

type RestaurantRepository interface {
	Create(
		ctx context.Context,
		tx Transaction,
		restaurant *Restaurant,
	) error

	GetByID(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
	) (*Restaurant, error)

	GetBySlug(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		slug string,
	) (*Restaurant, error)

	List(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		limit int,
		offset int,
	) ([]*Restaurant, error)

	Update(
		ctx context.Context,
		tx Transaction,
		restaurant *Restaurant,
	) error

	Activate(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
	) (*Restaurant, error)

	Deactivate(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
	) (*Restaurant, error)

	Suspend(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
	) (*Restaurant, error)
}
