package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	Raw() pgx.Tx
}

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
