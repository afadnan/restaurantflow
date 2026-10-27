package repository

import (
	"context"
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
)

// AddonRepository persists restaurant-scoped menu add-ons.
// All operations must execute inside the caller's transaction.
type AddonRepository interface {
	Create(
		ctx context.Context,
		tx sharedevents.Transaction,
		addon *domain.Addon,
	) error

	GetByID(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		addonID uuid.UUID,
	) (*domain.Addon, error)

	List(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		limit int,
		offset int,
	) ([]*domain.Addon, error)

	Update(
		ctx context.Context,
		tx sharedevents.Transaction,
		addon *domain.Addon,
	) (*domain.Addon, error)

	Deactivate(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		addonID uuid.UUID,
		updatedAt time.Time,
	) (*domain.Addon, error)
}
