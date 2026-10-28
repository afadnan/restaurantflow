package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

var ErrItemAddonUseCaseNotConfigured = errors.New(
	"menu item addon use case is not configured",
)

type ItemAddonUseCaseDependencies struct {
	DB         *postgres.DB
	ItemAddons repository.ItemAddonRepository
	Events     sharedevents.EventRepository
	Clock      Clock
}

func (d *ItemAddonUseCaseDependencies) now() time.Time {
	if d != nil && d.Clock != nil {
		return d.Clock.Now().UTC()
	}

	return SystemClock{}.Now().UTC()
}

type AttachItemAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	AddonID      uuid.UUID
	IsRequired   bool
	MaxQuantity  int32
}

type GetItemAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	AddonID      uuid.UUID
}

type ListItemAddonsInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	Limit        int
	Offset       int
}

type UpdateItemAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	AddonID      uuid.UUID
	IsRequired   bool
	MaxQuantity  int32
}

type DetachItemAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	AddonID      uuid.UUID
}

func itemAddonEvent(
	association *domain.ItemAddon,
	eventType sharedevents.EventType,
	occurredAt time.Time,
) domain.ItemAddonEvent {
	return domain.ItemAddonEvent{
		ID:          uuid.New(),
		Tenant:      association.TenantID,
		Restaurant:  association.RestaurantID,
		Item:        association.ItemID,
		Addon:       association.AddonID,
		IsRequired:  association.IsRequired,
		MaxQuantity: association.MaxQuantity,
		Occurred:    occurredAt.UTC(),
		Type:        eventType,
	}
}

type AttachItemAddonUseCase struct {
	ItemAddonUseCaseDependencies
}

func (uc *AttachItemAddonUseCase) Execute(
	ctx context.Context,
	input AttachItemAddonInput,
) (*domain.ItemAddon, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.ItemAddons == nil ||
		uc.Events == nil {
		return nil, ErrItemAddonUseCaseNotConfigured
	}

	association, err := domain.NewItemAddon(
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.AddonID,
		input.IsRequired,
		input.MaxQuantity,
	)
	if err != nil {
		return nil, err
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	persisted, err := uc.ItemAddons.Attach(ctx, tx, association)
	if err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, itemAddonEvent(
		persisted,
		domain.EventTypeItemAddonAttached,
		uc.now(),
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return persisted, nil
}

type GetItemAddonUseCase struct {
	ItemAddonUseCaseDependencies
}

func (uc *GetItemAddonUseCase) Execute(
	ctx context.Context,
	input GetItemAddonInput,
) (*domain.ItemAddon, error) {
	if uc == nil || uc.DB == nil || uc.ItemAddons == nil {
		return nil, ErrItemAddonUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	return uc.ItemAddons.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.AddonID,
	)
}

type ListItemAddonsUseCase struct {
	ItemAddonUseCaseDependencies
}

func (uc *ListItemAddonsUseCase) Execute(
	ctx context.Context,
	input ListItemAddonsInput,
) ([]*domain.ItemAddon, error) {
	if uc == nil || uc.DB == nil || uc.ItemAddons == nil {
		return nil, ErrItemAddonUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}
	if input.Limit < 0 || input.Offset < 0 {
		return nil, ErrInvalidPagination
	}

	limit := input.Limit
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	return uc.ItemAddons.List(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		limit,
		input.Offset,
	)
}

type UpdateItemAddonUseCase struct {
	ItemAddonUseCaseDependencies
}

func (uc *UpdateItemAddonUseCase) Execute(
	ctx context.Context,
	input UpdateItemAddonInput,
) (*domain.ItemAddon, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.ItemAddons == nil ||
		uc.Events == nil {
		return nil, ErrItemAddonUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	association, err := uc.ItemAddons.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.AddonID,
	)
	if err != nil {
		return nil, err
	}

	if err := association.Update(
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.AddonID,
		input.IsRequired,
		input.MaxQuantity,
	); err != nil {
		return nil, err
	}

	persisted, err := uc.ItemAddons.Update(ctx, tx, association)
	if err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, itemAddonEvent(
		persisted,
		domain.EventTypeItemAddonUpdated,
		uc.now(),
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return persisted, nil
}

type DetachItemAddonUseCase struct {
	ItemAddonUseCaseDependencies
}

func (uc *DetachItemAddonUseCase) Execute(
	ctx context.Context,
	input DetachItemAddonInput,
) (*domain.ItemAddon, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.ItemAddons == nil ||
		uc.Events == nil {
		return nil, ErrItemAddonUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	detached, err := uc.ItemAddons.Detach(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.AddonID,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, itemAddonEvent(
		detached,
		domain.EventTypeItemAddonDetached,
		uc.now(),
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return detached, nil
}
