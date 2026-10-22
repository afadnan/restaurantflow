package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresItemRepository struct{}

func NewPostgresItemRepository() *PostgresItemRepository {
	return &PostgresItemRepository{}
}

func (r *PostgresItemRepository) Create(
	ctx context.Context,
	tx sharedevents.Transaction,
	item *domain.Item,
) error {
	if item == nil {
		return domain.ErrInvalidItemInput
	}

	if err := validateItemIDs(
		item.TenantID,
		item.RestaurantID,
		item.ID,
	); err != nil {
		return err
	}

	if item.CategoryID == uuid.Nil {
		return domain.ErrInvalidCategoryID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	_, err = db.New(pgTx.Raw()).CreateMenuItem(
		ctx,
		db.CreateMenuItemParams{
			ItemID:       uuidToPgUUID(item.ID),
			TenantID:     uuidToPgUUID(item.TenantID),
			RestaurantID: uuidToPgUUID(item.RestaurantID),
			CategoryID:   uuidToPgUUID(item.CategoryID),
			Name:         item.Name,
			Description:  item.Description,
			Sku:          item.SKU,
			Price:        item.Price,
			Status:       string(item.Status),
			ImageUrl:     item.ImageURL,
			SortOrder:    item.SortOrder,
			IsVegetarian: item.IsVegetarian,
			IsVegan:      item.IsVegan,
			IsAvailable:  item.IsAvailable,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrCategoryNotFound
		}

		return fmt.Errorf("create menu item: %w", err)
	}

	return nil
}

func (r *PostgresItemRepository) GetByID(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
) (*domain.Item, error) {
	if err := validateItemIDs(tenantID, restaurantID, itemID); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).GetMenuItem(
		ctx,
		db.GetMenuItemParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}

		return nil, fmt.Errorf("get menu item: %w", err)
	}

	return mapItemFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.CategoryID,
		row.Name,
		row.Description,
		row.Sku,
		row.Price,
		row.Status,
		row.ImageUrl,
		row.SortOrder,
		row.IsVegetarian,
		row.IsVegan,
		row.IsAvailable,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func (r *PostgresItemRepository) List(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	limit int,
	offset int,
) ([]*domain.Item, error) {
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

	rows, err := db.New(pgTx.Raw()).ListMenuItems(
		ctx,
		db.ListMenuItemsParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			LimitCount:   int32(limit),
			OffsetCount:  int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list menu items: %w", err)
	}

	items := make([]*domain.Item, 0, len(rows))

	for _, row := range rows {
		item, err := mapItemFields(
			row.ID,
			row.TenantID,
			row.RestaurantID,
			row.CategoryID,
			row.Name,
			row.Description,
			row.Sku,
			row.Price,
			row.Status,
			row.ImageUrl,
			row.SortOrder,
			row.IsVegetarian,
			row.IsVegan,
			row.IsAvailable,
			row.CreatedAt,
			row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *PostgresItemRepository) Update(
	ctx context.Context,
	tx sharedevents.Transaction,
	item *domain.Item,
) error {
	if item == nil {
		return domain.ErrInvalidItemInput
	}

	if err := validateItemIDs(
		item.TenantID,
		item.RestaurantID,
		item.ID,
	); err != nil {
		return err
	}

	if item.CategoryID == uuid.Nil {
		return domain.ErrInvalidCategoryID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	_, err = db.New(pgTx.Raw()).UpdateMenuItem(
		ctx,
		db.UpdateMenuItemParams{
			TenantID:     uuidToPgUUID(item.TenantID),
			RestaurantID: uuidToPgUUID(item.RestaurantID),
			ItemID:       uuidToPgUUID(item.ID),
			CategoryID:   uuidToPgUUID(item.CategoryID),
			Name:         item.Name,
			Description:  item.Description,
			Sku:          item.SKU,
			Price:        item.Price,
			Status:       string(item.Status),
			ImageUrl:     item.ImageURL,
			SortOrder:    item.SortOrder,
			IsVegetarian: item.IsVegetarian,
			IsVegan:      item.IsVegan,
			IsAvailable:  item.IsAvailable,
		},
	)
	if err == nil {
		return nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("update menu item: %w", err)
	}

	// Determine whether the item is missing or its category is invalid.
	_, lookupErr := db.New(pgTx.Raw()).GetMenuItem(
		ctx,
		db.GetMenuItemParams{
			TenantID:     uuidToPgUUID(item.TenantID),
			RestaurantID: uuidToPgUUID(item.RestaurantID),
			ItemID:       uuidToPgUUID(item.ID),
		},
	)
	if errors.Is(lookupErr, pgx.ErrNoRows) {
		return domain.ErrItemNotFound
	}

	if lookupErr != nil {
		return fmt.Errorf("check menu item after update: %w", lookupErr)
	}

	return domain.ErrCategoryNotFound
}

func (r *PostgresItemRepository) Deactivate(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
) (*domain.Item, error) {
	if err := validateItemIDs(tenantID, restaurantID, itemID); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).DeactivateMenuItem(
		ctx,
		db.DeactivateMenuItemParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}

		return nil, fmt.Errorf("deactivate menu item: %w", err)
	}

	return mapItemFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.CategoryID,
		row.Name,
		row.Description,
		row.Sku,
		row.Price,
		row.Status,
		row.ImageUrl,
		row.SortOrder,
		row.IsVegetarian,
		row.IsVegan,
		row.IsAvailable,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func validateItemIDs(
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
) error {
	if tenantID == uuid.Nil {
		return domain.ErrInvalidTenantID
	}

	if restaurantID == uuid.Nil {
		return domain.ErrInvalidRestaurantID
	}

	if itemID == uuid.Nil {
		return domain.ErrInvalidItemID
	}

	return nil
}

func mapItemFields(
	id pgtype.UUID,
	tenantID pgtype.UUID,
	restaurantID pgtype.UUID,
	categoryID pgtype.UUID,
	name string,
	description string,
	sku string,
	price string,
	status string,
	imageURL string,
	sortOrder int32,
	isVegetarian bool,
	isVegan bool,
	isAvailable bool,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) (*domain.Item, error) {
	if !id.Valid || !tenantID.Valid || !restaurantID.Valid || !categoryID.Valid {
		return nil, fmt.Errorf("map menu item: database returned an invalid UUID")
	}

	if !createdAt.Valid || !updatedAt.Valid {
		return nil, fmt.Errorf("map menu item: database returned an invalid timestamp")
	}

	item := &domain.Item{
		ID:           pgUUIDToUUID(id),
		TenantID:     pgUUIDToUUID(tenantID),
		RestaurantID: pgUUIDToUUID(restaurantID),
		CategoryID:   pgUUIDToUUID(categoryID),
		Name:         name,
		Description:  description,
		SKU:          sku,
		Price:        price,
		Status:       domain.ItemStatus(status),
		ImageURL:     imageURL,
		SortOrder:    sortOrder,
		IsVegetarian: isVegetarian,
		IsVegan:      isVegan,
		IsAvailable:  isAvailable,
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}

	if item.ID == uuid.Nil {
		return nil, domain.ErrInvalidItemID
	}

	if item.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if item.RestaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	if item.CategoryID == uuid.Nil {
		return nil, domain.ErrInvalidCategoryID
	}

	return item, nil
}

var _ ItemRepository = (*PostgresItemRepository)(nil)
