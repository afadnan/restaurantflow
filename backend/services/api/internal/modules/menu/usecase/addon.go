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

var ErrAddonUseCaseNotConfigured = errors.New(
	"menu addon use case is not configured",
)

type AddonUseCaseDependencies struct {
	DB     *postgres.DB
	Addons repository.AddonRepository
	Events sharedevents.EventRepository
	Clock  Clock
}

func (d *AddonUseCaseDependencies) now() time.Time {
	if d != nil && d.Clock != nil {
		return d.Clock.Now().UTC()
	}
	return SystemClock{}.Now().UTC()
}

type CreateAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	Name         string
	Description  string
	Price        string
}

type GetAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	AddonID      uuid.UUID
}

type ListAddonsInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	Limit        int
	Offset       int
}

type UpdateAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	AddonID      uuid.UUID
	Name         string
	Description  string
	Price        string
}

type DeactivateAddonInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	AddonID      uuid.UUID
}

func addonEvent(
	addon *domain.Addon,
	eventType sharedevents.EventType,
	occurredAt time.Time,
) domain.AddonEvent {
	return domain.AddonEvent{
		ID:         uuid.New(),
		Tenant:     addon.TenantID,
		Restaurant: addon.RestaurantID,
		Addon:      addon.ID,
		Occurred:   occurredAt.UTC(),
		Type:       eventType,
	}
}

type CreateAddonUseCase struct {
	AddonUseCaseDependencies
}

func (uc *CreateAddonUseCase) Execute(
	ctx context.Context,
	input CreateAddonInput,
) (*domain.Addon, error) {
	if uc == nil || uc.DB == nil || uc.Addons == nil || uc.Events == nil {
		return nil, ErrAddonUseCaseNotConfigured
	}

	addon, err := domain.NewAddon(
		input.TenantID,
		input.RestaurantID,
		input.Name,
		input.Description,
		input.Price,
		uc.now(),
	)
	if err != nil {
		return nil, err
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := uc.Addons.Create(ctx, tx, addon); err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, addonEvent(
		addon,
		domain.EventTypeAddonCreated,
		addon.CreatedAt,
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return addon, nil
}

type GetAddonUseCase struct {
	AddonUseCaseDependencies
}

func (uc *GetAddonUseCase) Execute(
	ctx context.Context,
	input GetAddonInput,
) (*domain.Addon, error) {
	if uc == nil || uc.DB == nil || uc.Addons == nil {
		return nil, ErrAddonUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	return uc.Addons.GetByID(
		ctx, tx, input.TenantID, input.RestaurantID, input.AddonID,
	)
}

type ListAddonsUseCase struct {
	AddonUseCaseDependencies
}

func (uc *ListAddonsUseCase) Execute(
	ctx context.Context,
	input ListAddonsInput,
) ([]*domain.Addon, error) {
	if uc == nil || uc.DB == nil || uc.Addons == nil {
		return nil, ErrAddonUseCaseNotConfigured
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

	return uc.Addons.List(
		ctx, tx, input.TenantID, input.RestaurantID,
		limit, input.Offset,
	)
}

type UpdateAddonUseCase struct {
	AddonUseCaseDependencies
}

func (uc *UpdateAddonUseCase) Execute(
	ctx context.Context,
	input UpdateAddonInput,
) (*domain.Addon, error) {
	if uc == nil || uc.DB == nil || uc.Addons == nil || uc.Events == nil {
		return nil, ErrAddonUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	addon, err := uc.Addons.GetByID(
		ctx, tx, input.TenantID, input.RestaurantID, input.AddonID,
	)
	if err != nil {
		return nil, err
	}

	if err := addon.Update(
		input.TenantID,
		input.RestaurantID,
		input.AddonID,
		input.Name,
		input.Description,
		input.Price,
		uc.now(),
	); err != nil {
		return nil, err
	}

	persisted, err := uc.Addons.Update(ctx, tx, addon)
	if err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, addonEvent(
		persisted,
		domain.EventTypeAddonUpdated,
		persisted.UpdatedAt,
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return persisted, nil
}

type DeactivateAddonUseCase struct {
	AddonUseCaseDependencies
}

func (uc *DeactivateAddonUseCase) Execute(
	ctx context.Context,
	input DeactivateAddonInput,
) (*domain.Addon, error) {
	if uc == nil || uc.DB == nil || uc.Addons == nil || uc.Events == nil {
		return nil, ErrAddonUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	addon, err := uc.Addons.GetByID(
		ctx, tx, input.TenantID, input.RestaurantID, input.AddonID,
	)
	if err != nil {
		return nil, err
	}

	if err := addon.Deactivate(
		input.TenantID,
		input.RestaurantID,
		input.AddonID,
		uc.now(),
	); err != nil {
		return nil, err
	}

	persisted, err := uc.Addons.Deactivate(
		ctx, tx,
		input.TenantID,
		input.RestaurantID,
		input.AddonID,
		addon.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, addonEvent(
		persisted,
		domain.EventTypeAddonDeactivated,
		persisted.UpdatedAt,
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return persisted, nil
}
