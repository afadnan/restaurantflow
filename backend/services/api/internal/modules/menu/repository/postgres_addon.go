package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresAddonRepository struct{}

func NewPostgresAddonRepository() *PostgresAddonRepository {
	return &PostgresAddonRepository{}
}

func (r *PostgresAddonRepository) Create(
	ctx context.Context,
	tx sharedevents.Transaction,
	addon *domain.Addon,
) error {
	if addon == nil {
		return domain.ErrInvalidAddonInput
	}

	if err := validateAddonIDs(addon.TenantID, addon.RestaurantID, addon.ID); err != nil {
		return err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	_, err = db.New(pgTx.Raw()).CreateMenuAddon(
		ctx,
		db.CreateMenuAddonParams{
			AddonID:      uuidToPgUUID(addon.ID),
			Name:         addon.Name,
			Description:  addon.Description,
			Price:        addon.Price,
			TenantID:     uuidToPgUUID(addon.TenantID),
			RestaurantID: uuidToPgUUID(addon.RestaurantID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvalidRestaurantID
		}
		return fmt.Errorf("create menu addon: %w", err)
	}

	return nil
}

func (r *PostgresAddonRepository) GetByID(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	addonID uuid.UUID,
) (*domain.Addon, error) {
	if err := validateAddonIDs(tenantID, restaurantID, addonID); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).GetMenuAddonByID(
		ctx,
		db.GetMenuAddonByIDParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			AddonID:      uuidToPgUUID(addonID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAddonNotFound
		}
		return nil, fmt.Errorf("get menu addon: %w", err)
	}

	return mapAddonFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.Name,
		row.Description,
		row.Price,
		row.IsActive,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func (r *PostgresAddonRepository) List(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	limit int,
	offset int,
) ([]*domain.Addon, error) {
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

	rows, err := db.New(pgTx.Raw()).ListMenuAddons(
		ctx,
		db.ListMenuAddonsParams{
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			QueryLimit:   int32(limit),
			QueryOffset:  int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list menu addons: %w", err)
	}

	addons := make([]*domain.Addon, 0, len(rows))

	for _, row := range rows {
		addon, err := mapAddonFields(
			row.ID,
			row.TenantID,
			row.RestaurantID,
			row.Name,
			row.Description,
			row.Price,
			row.IsActive,
			row.CreatedAt,
			row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		addons = append(addons, addon)
	}

	return addons, nil
}

func (r *PostgresAddonRepository) Update(
	ctx context.Context,
	tx sharedevents.Transaction,
	addon *domain.Addon,
) (*domain.Addon, error) {
	if addon == nil {
		return nil, domain.ErrInvalidAddonInput
	}

	if err := validateAddonIDs(addon.TenantID, addon.RestaurantID, addon.ID); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).UpdateMenuAddon(
		ctx,
		db.UpdateMenuAddonParams{
			Name:        addon.Name,
			Description: addon.Description,
			Price:       addon.Price,
			UpdatedAt: pgtype.Timestamptz{
				Time:  addon.UpdatedAt.UTC(),
				Valid: true,
			},
			TenantID:     uuidToPgUUID(addon.TenantID),
			RestaurantID: uuidToPgUUID(addon.RestaurantID),
			AddonID:      uuidToPgUUID(addon.ID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAddonNotFound
		}
		return nil, fmt.Errorf("update menu addon: %w", err)
	}

	return mapAddonFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.Name,
		row.Description,
		row.Price,
		row.IsActive,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func (r *PostgresAddonRepository) Deactivate(
	ctx context.Context,
	tx sharedevents.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	addonID uuid.UUID,
	updatedAt time.Time,
) (*domain.Addon, error) {
	if err := validateAddonIDs(tenantID, restaurantID, addonID); err != nil {
		return nil, err
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	row, err := db.New(pgTx.Raw()).DeactivateMenuAddon(
		ctx,
		db.DeactivateMenuAddonParams{
			UpdatedAt: pgtype.Timestamptz{
				Time:  updatedAt.UTC(),
				Valid: true,
			},
			TenantID:     uuidToPgUUID(tenantID),
			RestaurantID: uuidToPgUUID(restaurantID),
			AddonID:      uuidToPgUUID(addonID),
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAddonNotFound
		}
		return nil, fmt.Errorf("deactivate menu addon: %w", err)
	}

	return mapAddonFields(
		row.ID,
		row.TenantID,
		row.RestaurantID,
		row.Name,
		row.Description,
		row.Price,
		row.IsActive,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func validateAddonIDs(
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
	addonID uuid.UUID,
) error {
	if tenantID == uuid.Nil {
		return domain.ErrInvalidTenantID
	}
	if restaurantID == uuid.Nil {
		return domain.ErrInvalidRestaurantID
	}
	if addonID == uuid.Nil {
		return domain.ErrInvalidAddonID
	}
	return nil
}

func mapAddonFields(
	id pgtype.UUID,
	tenantID pgtype.UUID,
	restaurantID pgtype.UUID,
	name string,
	description string,
	price string,
	isActive bool,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) (*domain.Addon, error) {
	if !id.Valid || !tenantID.Valid || !restaurantID.Valid {
		return nil, fmt.Errorf("map menu addon: invalid database UUID")
	}

	if !createdAt.Valid || !updatedAt.Valid {
		return nil, fmt.Errorf("map menu addon: invalid database timestamp")
	}

	return &domain.Addon{
		ID:           pgUUIDToUUID(id),
		TenantID:     pgUUIDToUUID(tenantID),
		RestaurantID: pgUUIDToUUID(restaurantID),
		Name:         name,
		Description:  description,
		Price:        price,
		IsActive:     isActive,
		CreatedAt:    createdAt.Time.UTC(),
		UpdatedAt:    updatedAt.Time.UTC(),
	}, nil
}

var _ AddonRepository = (*PostgresAddonRepository)(nil)
