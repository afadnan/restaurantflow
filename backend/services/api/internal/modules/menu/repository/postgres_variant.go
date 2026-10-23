package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresVariantRepository struct{}

func NewPostgresVariantRepository() *PostgresVariantRepository {
	return &PostgresVariantRepository{}
}

func (r *PostgresVariantRepository) Create(
	ctx context.Context,
	tx sharedevents.Transaction,
	variant *domain.Variant,
) error {
	if variant == nil {
		return domain.ErrInvalidVariantInput
	}

	if err := validateItemIDs(
		variant.TenantID,
		variant.RestaurantID,
		variant.ItemID,
	); err != nil {
		return err
	}

	if variant.ID == uuid.Nil {
		return domain.ErrInvalidVariantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	_, err = db.New(pgTx.Raw()).CreateMenuItemVariant(
		ctx,
		db.CreateMenuItemVariantParams{
			VariantID:    uuidToPgUUID(variant.ID),
			Name:         variant.Name,
			PriceDelta:   variant.PriceDelta,
			TenantID:     uuidToPgUUID(variant.TenantID),
			RestaurantID: uuidToPgUUID(variant.RestaurantID),
			ItemID:       uuidToPgUUID(variant.ItemID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrItemNotFound
		}

		return fmt.Errorf("create menu item variant: %w", err)
	}

	return nil
}

func (r *PostgresVariantRepository) GetByID(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
	variantID uuid.UUID,
) (*domain.Variant, error) {
	if err := validateItemIDs(tenantID, restaurantID, itemID); err != nil {
		return nil, err
	}

	if variantID == uuid.Nil {
		return nil, domain.ErrInvalidVariantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).GetMenuItemVariantByID(
		ctx,
		db.GetMenuItemVariantByIDParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
			VariantID:    uuidToPgUUID(variantID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVariantNotFound
		}

		return nil, fmt.Errorf("get menu item variant: %w", err)
	}

	return mapVariantFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.MenuItemID,
		row.Name,
		row.PriceDelta,
		row.IsActive,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func (r *PostgresVariantRepository) List(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
	limit int,
	offset int,
) ([]*domain.Variant, error) {
	if err := validateItemIDs(tenantID, restaurantID, itemID); err != nil {
		return nil, err
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

	rows, err := db.New(pgTx.Raw()).ListMenuItemVariants(
		ctx,
		db.ListMenuItemVariantsParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
			QueryLimit:   int32(limit),
			QueryOffset:  int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list menu item variants: %w", err)
	}

	variants := make([]*domain.Variant, 0, len(rows))

	for _, row := range rows {
		variant, err := mapVariantFields(
			row.ID,
			row.TenantID,
			row.RestaurantID,
			row.MenuItemID,
			row.Name,
			row.PriceDelta,
			row.IsActive,
			row.CreatedAt,
			row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		variants = append(variants, variant)
	}

	return variants, nil
}

func (r *PostgresVariantRepository) Update(
	ctx context.Context,
	tx sharedevents.Transaction,
	variant *domain.Variant,
) (*domain.Variant, error) {
	if variant == nil {
		return nil, domain.ErrInvalidVariantInput
	}

	if err := validateItemIDs(
		variant.TenantID,
		variant.RestaurantID,
		variant.ItemID,
	); err != nil {
		return nil, err
	}

	if variant.ID == uuid.Nil {
		return nil, domain.ErrInvalidVariantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).UpdateMenuItemVariant(
		ctx,
		db.UpdateMenuItemVariantParams{
			Name:       variant.Name,
			PriceDelta: variant.PriceDelta,
			UpdatedAt: pgtype.Timestamptz{
				Time:  variant.UpdatedAt.UTC(),
				Valid: true,
			},
			TenantID:     uuidToPgUUID(variant.TenantID),
			RestaurantID: uuidToPgUUID(variant.RestaurantID),
			ItemID:       uuidToPgUUID(variant.ItemID),
			VariantID:    uuidToPgUUID(variant.ID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVariantNotFound
		}

		return nil, fmt.Errorf("update menu item variant: %w", err)
	}

	return mapVariantFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.MenuItemID,
		row.Name,
		row.PriceDelta,
		row.IsActive,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func (r *PostgresVariantRepository) Deactivate(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
	variantID uuid.UUID,
	updatedAt time.Time,
) (*domain.Variant, error) {
	if err := validateItemIDs(tenantID, restaurantID, itemID); err != nil {
		return nil, err
	}

	if variantID == uuid.Nil {
		return nil, domain.ErrInvalidVariantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).DeactivateMenuItemVariant(
		ctx,
		db.DeactivateMenuItemVariantParams{
			UpdatedAt: pgtype.Timestamptz{
				Time:  updatedAt.UTC(),
				Valid: true,
			},
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
			VariantID:    uuidToPgUUID(variantID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVariantNotFound
		}

		return nil, fmt.Errorf("deactivate menu item variant: %w", err)
	}

	return mapVariantFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.MenuItemID,
		row.Name,
		row.PriceDelta,
		row.IsActive,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapVariantFields(
	id pgtype.UUID,
	tenantID pgtype.UUID,
	restaurantID pgtype.UUID,
	itemID pgtype.UUID,
	name string,
	priceDelta string,
	isActive bool,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) (*domain.Variant, error) {
	if !id.Valid ||
		!tenantID.Valid ||
		!restaurantID.Valid ||
		!itemID.Valid {
		return nil, fmt.Errorf("map menu item variant: invalid database UUID")
	}

	if !createdAt.Valid || !updatedAt.Valid {
		return nil, fmt.Errorf("map menu item variant: invalid database timestamp")
	}

	return &domain.Variant{
		ID:           pgUUIDToUUID(id),
		TenantID:     pgUUIDToUUID(tenantID),
		RestaurantID: pgUUIDToUUID(restaurantID),
		ItemID:       pgUUIDToUUID(itemID),
		Name:         name,
		PriceDelta:   priceDelta,
		IsActive:     isActive,
		CreatedAt:    createdAt.Time.UTC(),
		UpdatedAt:    updatedAt.Time.UTC(),
	}, nil
}

var _ VariantRepository = (*PostgresVariantRepository)(nil)

// Ensure the transaction helper's expected concrete type remains explicit.
var _ *postgres.Tx
