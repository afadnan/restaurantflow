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

var (
	ErrUseCaseNotConfigured = errors.New("menu use case is not configured")
	ErrInvalidPagination    = errors.New("invalid pagination")
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type CategoryUseCaseDependencies struct {
	DB         *postgres.DB
	Categories repository.CategoryRepository
	Events     sharedevents.EventRepository
	Clock      Clock
}

type CreateCategoryInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	Name         string
	Description  string
	SortOrder    int32
}

type GetCategoryInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	CategoryID   uuid.UUID
}

type ListCategoriesInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	Limit        int
	Offset       int
}

type UpdateCategoryInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	CategoryID   uuid.UUID
	Name         string
	Description  string
	SortOrder    int32
}

type DeactivateCategoryInput struct {
	TenantID     uuid.UUID
	RestaurantID uuid.UUID
	CategoryID   uuid.UUID
}

type CreateCategoryUseCase struct {
	CategoryUseCaseDependencies
}

func (uc *CreateCategoryUseCase) Execute(
	ctx context.Context,
	input CreateCategoryInput,
) (*domain.Category, error) {
	if uc == nil || uc.DB == nil || uc.Categories == nil || uc.Events == nil {
		return nil, ErrUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	now := uc.now()
	category, err := domain.NewCategory(domain.CreateCategoryInput{
		TenantID:     input.TenantID,
		RestaurantID: input.RestaurantID,
		Name:         input.Name,
		Description:  input.Description,
		SortOrder:    input.SortOrder,
	}, now)
	if err != nil {
		return nil, err
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := uc.Categories.Create(ctx, tx, category); err != nil {
		return nil, err
	}

	event := domain.CategoryEvent{
		ID:       uuid.New(),
		Tenant:   category.TenantID,
		Category: category.ID,
		Occurred: category.CreatedAt,
		Type:     domain.EventTypeCategoryCreated,
	}
	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return category, nil
}

type GetCategoryUseCase struct {
	CategoryUseCaseDependencies
}

func (uc *GetCategoryUseCase) Execute(
	ctx context.Context,
	input GetCategoryInput,
) (*domain.Category, error) {
	if uc == nil || uc.DB == nil || uc.Categories == nil {
		return nil, ErrUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	return uc.Categories.GetByID(
		ctx, tx, input.TenantID, input.RestaurantID, input.CategoryID,
	)
}

type ListCategoriesUseCase struct {
	CategoryUseCaseDependencies
}

func (uc *ListCategoriesUseCase) Execute(
	ctx context.Context,
	input ListCategoriesInput,
) ([]*domain.Category, error) {
	if uc == nil || uc.DB == nil || uc.Categories == nil {
		return nil, ErrUseCaseNotConfigured
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

	return uc.Categories.List(
		ctx, tx, input.TenantID, input.RestaurantID, limit, input.Offset,
	)
}

type UpdateCategoryUseCase struct {
	CategoryUseCaseDependencies
}

func (uc *UpdateCategoryUseCase) Execute(
	ctx context.Context,
	input UpdateCategoryInput,
) (*domain.Category, error) {
	if uc == nil || uc.DB == nil || uc.Categories == nil || uc.Events == nil {
		return nil, ErrUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	category, err := uc.Categories.GetByID(
		ctx, tx, input.TenantID, input.RestaurantID, input.CategoryID,
	)
	if err != nil {
		return nil, err
	}

	now := uc.now()
	if err := category.Update(domain.UpdateCategoryInput{
		TenantID:     input.TenantID,
		RestaurantID: input.RestaurantID,
		CategoryID:   input.CategoryID,
		Name:         input.Name,
		Description:  input.Description,
		SortOrder:    input.SortOrder,
	}, now); err != nil {
		return nil, err
	}

	if err := uc.Categories.Update(ctx, tx, category); err != nil {
		return nil, err
	}

	event := domain.CategoryEvent{
		ID:       uuid.New(),
		Tenant:   category.TenantID,
		Category: category.ID,
		Occurred: category.UpdatedAt,
		Type:     domain.EventTypeCategoryUpdated,
	}
	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return category, nil
}

type DeactivateCategoryUseCase struct {
	CategoryUseCaseDependencies
}

func (uc *DeactivateCategoryUseCase) Execute(
	ctx context.Context,
	input DeactivateCategoryInput,
) (*domain.Category, error) {
	if uc == nil || uc.DB == nil || uc.Categories == nil || uc.Events == nil {
		return nil, ErrUseCaseNotConfigured
	}
	if input.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	category, err := uc.Categories.Deactivate(
		ctx, tx, input.TenantID, input.RestaurantID, input.CategoryID,
	)
	if err != nil {
		return nil, err
	}

	event := domain.CategoryEvent{
		ID:       uuid.New(),
		Tenant:   category.TenantID,
		Category: category.ID,
		Occurred: category.UpdatedAt,
		Type:     domain.EventTypeCategoryDeactivated,
	}
	if err := uc.Events.Append(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return category, nil
}

func (uc *CreateCategoryUseCase) now() time.Time {
	if uc.Clock != nil {
		return uc.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (uc *UpdateCategoryUseCase) now() time.Time {
	if uc.Clock != nil {
		return uc.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
