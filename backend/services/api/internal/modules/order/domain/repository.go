package domain

import (
	"context"

	"github.com/google/uuid"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type OrderRepository interface {
	Create(
		ctx context.Context,
		tx Transaction,
		order *Order,
	) error

	GetByID(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		orderID uuid.UUID,
	) (*Order, error)

	UpdateState(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		orderID uuid.UUID,
		from OrderState,
		to OrderState,
	) error
}

type KDSRepository interface {
	Create(
		ctx context.Context,
		tx Transaction,
		kds *KDSOrder,
	) error

	GetByOrderID(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		orderID uuid.UUID,
	) (*KDSOrder, error)

	UpdateState(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		orderID uuid.UUID,
		from KDSState,
		to KDSState,
	) error
}

type InventoryRepository interface {
	DeductIngredientsForOrder(
		ctx context.Context,
		tx Transaction,
		tenantID uuid.UUID,
		order *Order,
	) error
}

type EventRepository interface {
	Append(
		ctx context.Context,
		tx Transaction,
		event sharedevents.DomainEvent,
	) error
}

type EventPublisher interface {
	Publish(
		ctx context.Context,
		event sharedevents.DomainEvent,
	) error
}

type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
