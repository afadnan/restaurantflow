package repository

import (
	"context"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
)

// ItemAddonRepository manages associations between menu items and add-ons.
// All operations must be scoped to a tenant and restaurant.
type ItemAddonRepository interface {
	Attach(
		ctx context.Context,
		tx sharedevents.Transaction,
		itemAddon *domain.ItemAddon,
	) (*domain.ItemAddon, error)

	GetByID(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
		addonID uuid.UUID,
	) (*domain.ItemAddon, error)

	List(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
		limit int,
		offset int,
	) ([]*domain.ItemAddon, error)

	Update(
		ctx context.Context,
		tx sharedevents.Transaction,
		itemAddon *domain.ItemAddon,
	) (*domain.ItemAddon, error)

	Detach(
		ctx context.Context,
		tx sharedevents.Transaction,
		tenantID uuid.UUID,
		restaurantID uuid.UUID,
		itemID uuid.UUID,
		addonID uuid.UUID,
	) (*domain.ItemAddon, error)
}
