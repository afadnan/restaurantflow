package repository

import (
	"context"
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
)

type VariantRepository interface {
	Create(
		ctx context.Context,
		tx sharedevents.Transaction,
		variant *domain.Variant,
	) error

	GetByID(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
		variantID uuid.UUID,
	) (*domain.Variant, error)

	List(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
		limit int,
		offset int,
	) ([]*domain.Variant, error)

	Update(
		ctx context.Context,
		tx sharedevents.Transaction,
		variant *domain.Variant,
	) (*domain.Variant, error)

	Deactivate(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
		variantID uuid.UUID,
		updatedAt time.Time,
	) (*domain.Variant, error)
}
