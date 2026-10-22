package repository

import (
	"context"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
)

type ItemRepository interface {
	Create(
		ctx context.Context,
		tx sharedevents.Transaction,
		item *domain.Item,
	) error

	GetByID(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
	) (*domain.Item, error)

	List(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		limit int,
		offset int,
	) ([]*domain.Item, error)

	Update(
		ctx context.Context,
		tx sharedevents.Transaction,
		item *domain.Item,
	) error

	Deactivate(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
	) (*domain.Item, error)
}
