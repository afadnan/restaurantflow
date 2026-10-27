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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresItemAddonRepository struct{}

func NewPostgresItemAddonRepository() *PostgresItemAddonRepository {
	return &PostgresItemAddonRepository{}
}

func (r *PostgresItemAddonRepository) Attach(
	ctx context.Context,
	tx sharedevents.Transaction,
	itemAddon *domain.ItemAddon,
) (*domain.ItemAddon, error) {
	if itemAddon == nil {
		return nil, domain.ErrInvalidItemAddonInput
	}

	if err := validateItemAddonIDs(
		itemAddon.TenantID,
		itemAddon.RestaurantID,
		itemAddon.ItemID,
		itemAddon.AddonID,
	); err != nil {
		return nil, err
	}

	if itemAddon.MaxQuantity <= 0 {
		return nil, domain.ErrInvalidAddonMaxQuantity
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).AttachMenuItemAddon(
		ctx,
		db.AttachMenuItemAddonParams{
			TenantID:     uuidToPgUUID(itemAddon.TenantID),
			RestaurantID: uuidToPgUUID(itemAddon.RestaurantID),
			ItemID:       uuidToPgUUID(itemAddon.ItemID),
			AddonID:      uuidToPgUUID(itemAddon.AddonID),
			IsRequired:   itemAddon.IsRequired,
			MaxQuantity:  itemAddon.MaxQuantity,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidItemAddonInput
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrItemAddonAlreadyExists
		}

		return nil, fmt.Errorf("attach menu item addon: %w", err)
	}

	return mapItemAddonFields(
		row.TenantID,
		row.RestaurantID,
		row.MenuItemID,
		row.AddonID,
		row.IsRequired,
		row.MaxQuantity,
	)
}

func (r *PostgresItemAddonRepository) GetByID(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
	addonID uuid.UUID,
) (*domain.ItemAddon, error) {
	if err := validateItemAddonIDs(
		tenantID, restaurantID, itemID, addonID,
	); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).GetMenuItemAddon(
		ctx,
		db.GetMenuItemAddonParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
			AddonID:      uuidToPgUUID(addonID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemAddonNotFound
		}
		return nil, fmt.Errorf("get menu item addon: %w", err)
	}

	return mapItemAddonFields(
		row.TenantID,
		row.RestaurantID,
		row.MenuItemID,
		row.AddonID,
		row.IsRequired,
		row.MaxQuantity,
	)
}

func (r *PostgresItemAddonRepository) List(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
	limit int,
	offset int,
) ([]*domain.ItemAddon, error) {
	if err := validateItemIDs(
		tenantID, restaurantID, itemID,
	); err != nil {
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

	rows, err := db.New(pgTx.Raw()).ListMenuItemAddons(
		ctx,
		db.ListMenuItemAddonsParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
			QueryLimit:   int32(limit),
			QueryOffset:  int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list menu item addons: %w", err)
	}

	itemAddons := make([]*domain.ItemAddon, 0, len(rows))

	for _, row := range rows {
		itemAddon, err := mapItemAddonFields(
			row.TenantID,
			row.RestaurantID,
			row.MenuItemID,
			row.AddonID,
			row.IsRequired,
			row.MaxQuantity,
		)
		if err != nil {
			return nil, err
		}

		itemAddons = append(itemAddons, itemAddon)
	}

	return itemAddons, nil
}

func (r *PostgresItemAddonRepository) Update(
	ctx context.Context,
	tx sharedevents.Transaction,
	itemAddon *domain.ItemAddon,
) (*domain.ItemAddon, error) {
	if itemAddon == nil {
		return nil, domain.ErrInvalidItemAddonInput
	}

	if err := validateItemAddonIDs(
		itemAddon.TenantID,
		itemAddon.RestaurantID,
		itemAddon.ItemID,
		itemAddon.AddonID,
	); err != nil {
		return nil, err
	}

	if itemAddon.MaxQuantity <= 0 {
		return nil, domain.ErrInvalidAddonMaxQuantity
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).UpdateMenuItemAddon(
		ctx,
		db.UpdateMenuItemAddonParams{
			TenantID:     uuidToPgUUID(itemAddon.TenantID),
			RestaurantID: uuidToPgUUID(itemAddon.RestaurantID),
			ItemID:       uuidToPgUUID(itemAddon.ItemID),
			AddonID:      uuidToPgUUID(itemAddon.AddonID),
			IsRequired:   itemAddon.IsRequired,
			MaxQuantity:  itemAddon.MaxQuantity,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemAddonNotFound
		}
		return nil, fmt.Errorf("update menu item addon: %w", err)
	}

	return mapItemAddonFields(
		row.TenantID,
		row.RestaurantID,
		row.MenuItemID,
		row.AddonID,
		row.IsRequired,
		row.MaxQuantity,
	)
}

func (r *PostgresItemAddonRepository) Detach(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
	addonID uuid.UUID,
) (*domain.ItemAddon, error) {
	if err := validateItemAddonIDs(
		tenantID, restaurantID, itemID, addonID,
	); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).DetachMenuItemAddon(
		ctx,
		db.DetachMenuItemAddonParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			ItemID:       uuidToPgUUID(itemID),
			AddonID:      uuidToPgUUID(addonID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemAddonNotFound
		}
		return nil, fmt.Errorf("detach menu item addon: %w", err)
	}

	return mapItemAddonFields(
		row.TenantID,
		row.RestaurantID,
		row.MenuItemID,
		row.AddonID,
		row.IsRequired,
		row.MaxQuantity,
	)
}

func validateItemAddonIDs(
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	itemID uuid.UUID,
	addonID uuid.UUID,
) error {
	if err := validateItemIDs(
		tenantID, restaurantID, itemID,
	); err != nil {
		return err
	}

	if addonID == uuid.Nil {
		return domain.ErrInvalidAddonID
	}

	return nil
}

func mapItemAddonFields(
	tenantID pgtype.UUID,
	restaurantID pgtype.UUID,
	itemID pgtype.UUID,
	addonID pgtype.UUID,
	isRequired bool,
	maxQuantity int32,
) (*domain.ItemAddon, error) {
	if !tenantID.Valid ||
		!restaurantID.Valid ||
		!itemID.Valid ||
		!addonID.Valid {
		return nil, fmt.Errorf(
			"map menu item addon: invalid database UUID",
		)
	}

	if maxQuantity <= 0 {
		return nil, fmt.Errorf(
			"map menu item addon: invalid max quantity",
		)
	}

	return &domain.ItemAddon{
		TenantID:     pgUUIDToUUID(tenantID),
		RestaurantID: pgUUIDToUUID(restaurantID),
		ItemID:       pgUUIDToUUID(itemID),
		AddonID:      pgUUIDToUUID(addonID),
		IsRequired:   isRequired,
		MaxQuantity:  maxQuantity,
	}, nil
}

var _ ItemAddonRepository = (*PostgresItemAddonRepository)(nil)
