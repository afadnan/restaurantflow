package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type PostgresCategoryRepository struct{}

func NewPostgresCategoryRepository() *PostgresCategoryRepository {
	return &PostgresCategoryRepository{}
}

func (r *PostgresCategoryRepository) Create(
	ctx context.Context,
	tx Transaction,
	category *domain.Category,
) error {
	if category == nil {
		return domain.ErrInvalidCategoryInput
	}

	if category.ID == uuid.Nil {
		return domain.ErrInvalidCategoryID
	}

	if category.TenantID == uuid.Nil {
		return domain.ErrInvalidTenantID
	}

	if category.RestaurantID == uuid.Nil {
		return domain.ErrInvalidRestaurantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	q := db.New(pgTx.Raw())

	_, err = q.CreateMenuCategory(ctx, db.CreateMenuCategoryParams{
		CategoryID:   uuidToPgUUID(category.ID),
		TenantID:     uuidToPgUUID(category.TenantID),
		RestaurantID: uuidToPgUUID(category.RestaurantID),
		Name:         category.Name,
		Description:  textToPgText(category.Description),
		SortOrder:    category.SortOrder,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvalidRestaurantID
		}

		return fmt.Errorf("create menu category: %w", err)
	}

	return nil
}

func (r *PostgresCategoryRepository) GetByID(
	ctx context.Context,
	tx Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	categoryID uuid.UUID,
) (*domain.Category, error) {
	if err := validateCategoryIDs(tenantID, restaurantID, categoryID); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).GetMenuCategoryByID(
		ctx,
		db.GetMenuCategoryByIDParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ID:           uuidToPgUUID(categoryID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}

		return nil, fmt.Errorf("get menu category: %w", err)
	}

	return mapCategory(row)
}

func (r *PostgresCategoryRepository) List(
	ctx context.Context,
	tx Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	limit int,
	offset int,
) ([]*domain.Category, error) {
	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if restaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	rows, err := db.New(pgTx.Raw()).ListMenuCategories(
		ctx,
		db.ListMenuCategoriesParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			Limit:        int32(limit),
			Offset:       int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list menu categories: %w", err)
	}

	categories := make([]*domain.Category, 0, len(rows))
	for _, row := range rows {
		category, err := mapCategory(row)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}

func (r *PostgresCategoryRepository) Update(
	ctx context.Context,
	tx Transaction,
	category *domain.Category,
) error {
	if category == nil {
		return domain.ErrInvalidCategoryInput
	}

	if err := validateCategoryIDs(
		category.TenantID,
		category.RestaurantID,
		category.ID,
	); err != nil {
		return err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	_, err = db.New(pgTx.Raw()).UpdateMenuCategory(
		ctx,
		db.UpdateMenuCategoryParams{
			TenantID:     uuidToPgUUID(category.TenantID),
			RestaurantID: uuidToPgUUID(category.RestaurantID),
			ID:           uuidToPgUUID(category.ID),
			Name:         category.Name,
			Description:  textToPgText(category.Description),
			SortOrder:    category.SortOrder,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrCategoryNotFound
		}
		return fmt.Errorf("update menu category: %w", err)
	}

	return nil
}

func (r *PostgresCategoryRepository) Deactivate(
	ctx context.Context,
	tx Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	categoryID uuid.UUID,
) (*domain.Category, error) {
	if err := validateCategoryIDs(tenantID, restaurantID, categoryID); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).DeactivateMenuCategory(
		ctx,
		db.DeactivateMenuCategoryParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ID:           uuidToPgUUID(categoryID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("deactivate menu category: %w", err)
	}

	return mapCategory(row)
}

func postgresTx(tx Transaction) (*postgres.Tx, error) {
	if tx == nil {
		return nil, errors.New("transaction is required")
	}

	pgTx, ok := tx.(*postgres.Tx)
	if !ok {
		return nil, errors.New("transaction is not a postgres transaction")
	}

	return pgTx, nil
}

func validateCategoryIDs(
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	categoryID uuid.UUID,
) error {
	if tenantID == uuid.Nil {
		return domain.ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil {
		return domain.ErrInvalidRestaurantID
	}
	if categoryID == uuid.Nil {
		return domain.ErrInvalidCategoryID
	}
	return nil
}

func mapCategory(row db.MenuCategory) (*domain.Category, error) {
	category := &domain.Category{
		ID:           pgUUIDToUUID(row.ID),
		TenantID:     pgUUIDToUUID(row.TenantID),
		RestaurantID: pgUUIDToUUID(row.RestaurantID),
		Name:         row.Name,
		Description:  pgTextToString(row.Description),
		SortOrder:    row.SortOrder,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}

	if category.ID == uuid.Nil {
		return nil, domain.ErrInvalidCategoryID
	}
	if category.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}
	if category.RestaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	return category, nil
}

func uuidToPgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: value,
		Valid: value != uuid.Nil,
	}
}

func pgUUIDToUUID(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return value.Bytes
}

func textToPgText(value string) pgtype.Text {
	return pgtype.Text{
		String: value,
		Valid:  value != "",
	}
}

func pgTextToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

var _ CategoryRepository = (*PostgresCategoryRepository)(nil)
