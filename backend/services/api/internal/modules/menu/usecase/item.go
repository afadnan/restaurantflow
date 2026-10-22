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

var ErrItemUseCaseNotConfigured = errors.New("menu item use case is not configured")

type ItemUseCaseDependencies struct {
	DB     *postgres.DB
	Items  repository.ItemRepository
	Events sharedevents.EventRepository
	Clock  Clock
}

type CreateItemInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	CategoryID   uuid.UUID
	Name         string
	Description  string
	SKU          string
	Price        string
	Status       domain.ItemStatus
	ImageURL     string
	SortOrder    int32
	IsVegetarian bool
	IsVegan      bool
	IsAvailable  bool
}

type GetItemInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
}

type ListItemsInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	Limit        int
	Offset       int
}

type UpdateItemInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
	CategoryID   uuid.UUID
	Name         string
	Description  string
	SKU          string
	Price        string
	Status       domain.ItemStatus
	ImageURL     string
	SortOrder    int32
	IsVegetarian bool
	IsVegan      bool
	IsAvailable  bool
}

type DeactivateItemInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	ItemID       uuid.UUID
}

type CreateItemUseCase struct {
	ItemUseCaseDependencies
}

func (uc *CreateItemUseCase) Execute(
	ctx context.Context,
	input CreateItemInput,
) (*domain.Item, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.Items == nil ||
		uc.Events == nil {
		return nil, ErrItemUseCaseNotConfigured
	}

	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	item, err := domain.NewItem(domain.CreateItemInput{
		TenantID:     input.TenantID,
		RestaurantID: input.RestaurantID,
		CategoryID:   input.CategoryID,
		Name:         input.Name,
		Description:  input.Description,
		SKU:          input.SKU,
		Price:        input.Price,
		Status:       input.Status,
		ImageURL:     input.ImageURL,
		SortOrder:    input.SortOrder,
		IsVegetarian: input.IsVegetarian,
		IsVegan:      input.IsVegan,
		IsAvailable:  input.IsAvailable,
	}, uc.now())
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

	if err := uc.Items.Create(ctx, tx, item); err != nil {
		return nil, err
	}

	event := domain.ItemEvent{
		ID:       uuid.New(),
		Tenant:   item.TenantID,
		Item:     item.ID,
		Occurred: item.CreatedAt,
		Type:     domain.EventTypeItemCreated,
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return item, nil
}

type GetItemUseCase struct {
	ItemUseCaseDependencies
}

func (uc *GetItemUseCase) Execute(
	ctx context.Context,
	input GetItemInput,
) (*domain.Item, error) {
	if uc == nil || uc.DB == nil || uc.Items == nil {
		return nil, ErrItemUseCaseNotConfigured
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

	return uc.Items.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
	)
}

type ListItemsUseCase struct {
	ItemUseCaseDependencies
}

func (uc *ListItemsUseCase) Execute(
	ctx context.Context,
	input ListItemsInput,
) ([]*domain.Item, error) {
	if uc == nil || uc.DB == nil || uc.Items == nil {
		return nil, ErrItemUseCaseNotConfigured
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

	return uc.Items.List(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		limit,
		input.Offset,
	)
}

type UpdateItemUseCase struct {
	ItemUseCaseDependencies
}

func (uc *UpdateItemUseCase) Execute(
	ctx context.Context,
	input UpdateItemInput,
) (*domain.Item, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.Items == nil ||
		uc.Events == nil {
		return nil, ErrItemUseCaseNotConfigured
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

	item, err := uc.Items.GetByID(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
	)
	if err != nil {
		return nil, err
	}

	now := uc.now()

	if err := item.Update(domain.UpdateItemInput{
		TenantID:     input.TenantID,
		RestaurantID: input.RestaurantID,
		ItemID:       input.ItemID,
		CategoryID:   input.CategoryID,
		Name:         input.Name,
		Description:  input.Description,
		SKU:          input.SKU,
		Price:        input.Price,
		Status:       input.Status,
		ImageURL:     input.ImageURL,
		SortOrder:    input.SortOrder,
		IsVegetarian: input.IsVegetarian,
		IsVegan:      input.IsVegan,
		IsAvailable:  input.IsAvailable,
	}, now); err != nil {
		return nil, err
	}

	if err := uc.Items.Update(ctx, tx, item); err != nil {
		return nil, err
	}

	event := domain.ItemEvent{
		ID:       uuid.New(),
		Tenant:   item.TenantID,
		Item:     item.ID,
		Occurred: item.UpdatedAt,
		Type:     domain.EventTypeItemUpdated,
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return item, nil
}

type DeactivateItemUseCase struct {
	ItemUseCaseDependencies
}

func (uc *DeactivateItemUseCase) Execute(
	ctx context.Context,
	input DeactivateItemInput,
) (*domain.Item, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.Items == nil ||
		uc.Events == nil {
		return nil, ErrItemUseCaseNotConfigured
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

	item, err := uc.Items.Deactivate(
		ctx,
		tx,
		input.TenantID,
		input.RestaurantID,
		input.ItemID,
	)
	if err != nil {
		return nil, err
	}

	event := domain.ItemEvent{
		ID:       uuid.New(),
		Tenant:   item.TenantID,
		Item:     item.ID,
		Occurred: item.UpdatedAt,
		Type:     domain.EventTypeItemDeactivated,
	}

	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return item, nil
}

func (uc *CreateItemUseCase) now() time.Time {
	if uc.Clock != nil {
		return uc.Clock.Now().UTC()
	}

	return time.Now().UTC()
}

func (uc *UpdateItemUseCase) now() time.Time {
	if uc.Clock != nil {
		return uc.Clock.Now().UTC()
	}

	return time.Now().UTC()
}
