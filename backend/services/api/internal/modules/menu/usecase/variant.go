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

var ErrVariantUseCaseNotConfigured = errors.New(
	"menu variant use case is not configured",
)

type VariantUseCaseDependencies struct {
	DB       *postgres.DB
	Variants repository.VariantRepository
	Events   sharedevents.EventRepository
	Clock    Clock
}

func (d *VariantUseCaseDependencies) now() time.Time {
	if d != nil && d.Clock != nil {
		return d.Clock.Now().UTC()
	}

	return SystemClock{}.Now().UTC()
}

type CreateVariantInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	Name         string
	PriceDelta   string
}

type GetVariantInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	VariantID    uuid.UUID
}

type ListVariantsInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	Limit        int
	Offset       int
}

type UpdateVariantInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	VariantID    uuid.UUID
	Name         string
	PriceDelta   string
}

type DeactivateVariantInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	VariantID    uuid.UUID
}

type CreateVariantUseCase struct {
	VariantUseCaseDependencies
}

func (uc *CreateVariantUseCase) Execute(
	ctx context.Context,
	input CreateVariantInput,
) (*domain.Variant, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.Variants == nil ||
		uc.Events == nil {
		return nil, ErrVariantUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	variant, err := domain.NewVariant(
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.Name,
		input.PriceDelta,
		uc.now(),
	)
	if err != nil {
		return nil, err
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := uc.Variants.Create(ctx, tx, variant); err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, VariantEvent(
		variant,
		domain.EventTypeVariantCreated,
		variant.CreatedAt,
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return variant, nil
}

type GetVariantUseCase struct {
	VariantUseCaseDependencies
}

func (uc *GetVariantUseCase) Execute(
	ctx context.Context,
	input GetVariantInput,
) (*domain.Variant, error) {
	if uc == nil || uc.DB == nil || uc.Variants == nil {
		return nil, ErrVariantUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	return uc.Variants.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.VariantID,
	)
}

type ListVariantsUseCase struct {
	VariantUseCaseDependencies
}

func (uc *ListVariantsUseCase) Execute(
	ctx context.Context,
	input ListVariantsInput,
) ([]*domain.Variant, error) {
	if uc == nil || uc.DB == nil || uc.Variants == nil {
		return nil, ErrVariantUseCaseNotConfigured
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
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	return uc.Variants.List(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		limit,
		input.Offset,
	)
}

type UpdateVariantUseCase struct {
	VariantUseCaseDependencies
}

func (uc *UpdateVariantUseCase) Execute(
	ctx context.Context,
	input UpdateVariantInput,
) (*domain.Variant, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.Variants == nil ||
		uc.Events == nil {
		return nil, ErrVariantUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	variant, err := uc.Variants.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.VariantID,
	)
	if err != nil {
		return nil, err
	}

	if err := variant.Update(
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.VariantID,
		input.Name,
		input.PriceDelta,
		uc.now(),
	); err != nil {
		return nil, err
	}

	persisted, err := uc.Variants.Update(ctx, tx, variant)
	if err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, VariantEvent(
		persisted,
		domain.EventTypeVariantUpdated,
		persisted.UpdatedAt,
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return persisted, nil
}

type DeactivateVariantUseCase struct {
	VariantUseCaseDependencies
}

func (uc *DeactivateVariantUseCase) Execute(
	ctx context.Context,
	input DeactivateVariantInput,
) (*domain.Variant, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.Variants == nil ||
		uc.Events == nil {
		return nil, ErrVariantUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	variant, err := uc.Variants.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.VariantID,
	)
	if err != nil {
		return nil, err
	}

	if err := variant.Deactivate(
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.VariantID,
		uc.now(),
	); err != nil {
		return nil, err
	}

	persisted, err := uc.Variants.Deactivate(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
		input.VariantID,
		variant.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.Events.Append(ctx, tx, VariantEvent(
		persisted,
		domain.EventTypeVariantDeactivated,
		persisted.UpdatedAt,
	)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return persisted, nil
}

func VariantEvent(
	variant *domain.Variant,
	eventType sharedevents.EventType,
	occurredAt time.Time,
) domain.VariantEvent {
	return domain.VariantEvent{
		ID:       uuid.New(),
		Tenant:   variant.TenantID,
		Variant:  variant.ID,
		Occurred: occurredAt.UTC(),
		Type:     eventType,
	}
}
