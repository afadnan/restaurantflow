package repository

import (
	"context"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
)

type Transaction = sharedevents.Transaction

type CategoryRepository interface {
	Create(
		ctx context.Context,
		tx Transaction,
		category *domain.Category,
	) error

	GetByID(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		categoryID uuid.UUID,
	) (*domain.Category, error)

	List(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		limit int,
		offset int,
	) ([]*domain.Category, error)

	Update(
		ctx context.Context,
		tx Transaction,
		category *domain.Category,
	) error

	Deactivate(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		categoryID uuid.UUID,
	) (*domain.Category, error)
}
