package repository

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/restaurant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type PostgresRestaurantRepository struct{}

func NewPostgresRestaurantRepository() *PostgresRestaurantRepository {
	return &PostgresRestaurantRepository{}
}

func (r *PostgresRestaurantRepository) Create(
	ctx context.Context,
	tx domain.Transaction,
	restaurant *domain.Restaurant,
) error {
	if restaurant == nil {
		return errors.New("restaurant is required")
	}

	if restaurant.ID.UUID() == uuid.Nil {
		return domain.ErrInvalidRestaurantID
	}

	if restaurant.TenantID.UUID() == uuid.Nil {
		return domain.ErrInvalidTenantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	latitude, err := floatToNumeric(restaurant.Latitude)
	if err != nil {
		return fmt.Errorf("convert latitude: %w", err)
	}

	longitude, err := floatToNumeric(restaurant.Longitude)
	if err != nil {
		return fmt.Errorf("convert longitude: %w", err)
	}

	q := db.New(pgTx.Raw())

	_, err = q.CreateRestaurant(ctx, db.CreateRestaurantParams{
		ID:           uuidToPgUUID(restaurant.ID.UUID()),
		TenantID:     uuidToPgUUID(restaurant.TenantID.UUID()),
		Name:         restaurant.Name.String(),
		Slug:         restaurant.Slug.String(),
		Description:  textToPgText(restaurant.Description),
		Phone:        textToPgText(restaurant.Phone),
		Email:        textToPgText(restaurant.Email),
		AddressLine1: textToPgText(restaurant.AddressLine1),
		AddressLine2: textToPgText(restaurant.AddressLine2),
		City:         textToPgText(restaurant.City),
		State:        textToPgText(restaurant.State),
		PostalCode:   textToPgText(restaurant.PostalCode),
		Country:      restaurant.Country,
		Latitude:     latitude,
		Longitude:    longitude,
		Status:       db.RestaurantStatus(restaurant.Status),
	})
	if err != nil {
		return fmt.Errorf("create restaurant: %w", err)
	}

	return nil
}

func (r *PostgresRestaurantRepository) GetByID(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
) (*domain.Restaurant, error) {
	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if restaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	q := db.New(pgTx.Raw())

	row, err := q.GetRestaurantByID(ctx, db.GetRestaurantByIDParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       uuidToPgUUID(restaurantID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRestaurantNotFound
		}

		return nil, fmt.Errorf("get restaurant by id: %w", err)
	}

	return mapRestaurant(row)
}

func (r *PostgresRestaurantRepository) GetBySlug(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	slug string,
) (*domain.Restaurant, error) {
	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if slug == "" {
		return nil, domain.ErrInvalidRestaurantSlug
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	q := db.New(pgTx.Raw())

	row, err := q.GetRestaurantBySlug(ctx, db.GetRestaurantBySlugParams{
		TenantID: uuidToPgUUID(tenantID),
		Slug:     slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRestaurantNotFound
		}

		return nil, fmt.Errorf("get restaurant by slug: %w", err)
	}

	return mapRestaurant(row)
}

func (r *PostgresRestaurantRepository) List(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	limit int,
	offset int,
) ([]*domain.Restaurant, error) {
	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
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

	q := db.New(pgTx.Raw())

	rows, err := q.ListRestaurants(ctx, db.ListRestaurantsParams{
		TenantID: uuidToPgUUID(tenantID),
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list restaurants: %w", err)
	}

	restaurants := make([]*domain.Restaurant, 0, len(rows))

	for _, row := range rows {
		restaurant, err := mapRestaurant(row)
		if err != nil {
			return nil, err
		}

		restaurants = append(restaurants, restaurant)
	}

	return restaurants, nil
}

func (r *PostgresRestaurantRepository) Update(
	ctx context.Context,
	tx domain.Transaction,
	restaurant *domain.Restaurant,
) error {
	if restaurant == nil {
		return errors.New("restaurant is required")
	}

	if restaurant.ID.UUID() == uuid.Nil {
		return domain.ErrInvalidRestaurantID
	}

	if restaurant.TenantID.UUID() == uuid.Nil {
		return domain.ErrInvalidTenantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	latitude, err := floatToNumeric(restaurant.Latitude)
	if err != nil {
		return fmt.Errorf("convert latitude: %w", err)
	}

	longitude, err := floatToNumeric(restaurant.Longitude)
	if err != nil {
		return fmt.Errorf("convert longitude: %w", err)
	}

	q := db.New(pgTx.Raw())

	_, err = q.UpdateRestaurant(ctx, db.UpdateRestaurantParams{
		TenantID:     uuidToPgUUID(restaurant.TenantID.UUID()),
		ID:           uuidToPgUUID(restaurant.ID.UUID()),
		Name:         restaurant.Name.String(),
		Slug:         restaurant.Slug.String(),
		Description:  textToPgText(restaurant.Description),
		Phone:        textToPgText(restaurant.Phone),
		Email:        textToPgText(restaurant.Email),
		AddressLine1: textToPgText(restaurant.AddressLine1),
		AddressLine2: textToPgText(restaurant.AddressLine2),
		City:         textToPgText(restaurant.City),
		State:        textToPgText(restaurant.State),
		PostalCode:   textToPgText(restaurant.PostalCode),
		Country:      restaurant.Country,
		Latitude:     latitude,
		Longitude:    longitude,
	})
	if err != nil {
		return fmt.Errorf("update restaurant: %w", err)
	}

	return nil
}

func (r *PostgresRestaurantRepository) Activate(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
) (*domain.Restaurant, error) {
	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if restaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	q := db.New(pgTx.Raw())

	row, err := q.ActivateRestaurant(ctx, db.ActivateRestaurantParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       uuidToPgUUID(restaurantID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRestaurantNotFound
		}

		return nil, fmt.Errorf("activate restaurant: %w", err)
	}

	return mapRestaurant(row)
}

func (r *PostgresRestaurantRepository) Deactivate(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
) (*domain.Restaurant, error) {
	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if restaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	q := db.New(pgTx.Raw())

	row, err := q.DeactivateRestaurant(ctx, db.DeactivateRestaurantParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       uuidToPgUUID(restaurantID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRestaurantNotFound
		}

		return nil, fmt.Errorf("deactivate restaurant: %w", err)
	}

	return mapRestaurant(row)
}

func (r *PostgresRestaurantRepository) Suspend(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	restaurantID uuid.UUID,
) (*domain.Restaurant, error) {
	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if restaurantID == uuid.Nil {
		return nil, domain.ErrInvalidRestaurantID
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	q := db.New(pgTx.Raw())

	row, err := q.SuspendRestaurant(ctx, db.SuspendRestaurantParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       uuidToPgUUID(restaurantID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRestaurantNotFound
		}

		return nil, fmt.Errorf("suspend restaurant: %w", err)
	}

	return mapRestaurant(row)
}

func postgresTx(tx domain.Transaction) (*postgres.Tx, error) {
	if tx == nil {
		return nil, errors.New("transaction is required")
	}

	pgTx, ok := tx.(*postgres.Tx)
	if !ok {
		return nil, errors.New("transaction is not a postgres transaction")
	}

	return pgTx, nil
}

func mapRestaurant(row db.Restaurant) (*domain.Restaurant, error) {
	id, err := domain.NewRestaurantIDFromUUID(pgUUIDToUUID(row.ID))
	if err != nil {
		return nil, fmt.Errorf("map restaurant id: %w", err)
	}

	tenantID, err := domain.NewTenantIDFromUUID(pgUUIDToUUID(row.TenantID))
	if err != nil {
		return nil, fmt.Errorf("map tenant id: %w", err)
	}

	name, err := domain.NewRestaurantName(row.Name)
	if err != nil {
		return nil, fmt.Errorf("map restaurant name: %w", err)
	}

	slug, err := domain.NewRestaurantSlug(row.Slug)
	if err != nil {
		return nil, fmt.Errorf("map restaurant slug: %w", err)
	}

	status := domain.RestaurantStatus(row.Status)
	if !status.Valid() {
		return nil, fmt.Errorf(
			"map restaurant status: %w",
			domain.ErrInvalidRestaurantStatus,
		)
	}

	latitude, err := numericToFloat(row.Latitude)
	if err != nil {
		return nil, fmt.Errorf("map restaurant latitude: %w", err)
	}

	longitude, err := numericToFloat(row.Longitude)
	if err != nil {
		return nil, fmt.Errorf("map restaurant longitude: %w", err)
	}

	if err := validateMappedCoordinates(latitude, longitude); err != nil {
		return nil, err
	}

	return &domain.Restaurant{
		ID:           id,
		TenantID:     tenantID,
		Name:         name,
		Slug:         slug,
		Description:  pgTextToString(row.Description),
		Phone:        pgTextToString(row.Phone),
		Email:        pgTextToString(row.Email),
		AddressLine1: pgTextToString(row.AddressLine1),
		AddressLine2: pgTextToString(row.AddressLine2),
		City:         pgTextToString(row.City),
		State:        pgTextToString(row.State),
		PostalCode:   pgTextToString(row.PostalCode),
		Country:      row.Country,
		Latitude:     latitude,
		Longitude:    longitude,
		Status:       status,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

func validateMappedCoordinates(
	latitude *float64,
	longitude *float64,
) error {
	if latitude != nil && (*latitude < -90 || *latitude > 90) {
		return domain.ErrInvalidLatitude
	}

	if longitude != nil && (*longitude < -180 || *longitude > 180) {
		return domain.ErrInvalidLongitude
	}

	return nil
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

func floatToNumeric(value *float64) (pgtype.Numeric, error) {
	if value == nil {
		return pgtype.Numeric{}, nil
	}

	text := strconv.FormatFloat(*value, 'f', 7, 64)

	var numeric pgtype.Numeric
	if err := numeric.Scan(text); err != nil {
		return pgtype.Numeric{}, fmt.Errorf(
			"parse numeric %q: %w",
			text,
			err,
		)
	}

	return numeric, nil
}

func numericToFloat(value pgtype.Numeric) (*float64, error) {
	if !value.Valid {
		return nil, nil
	}

	if value.Int == nil {
		return nil, errors.New("numeric value has no integer representation")
	}

	rat := new(big.Rat).SetInt(value.Int)

	if value.Exp > 0 {
		scale := new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(int64(value.Exp)),
			nil,
		)

		rat.Mul(
			rat,
			new(big.Rat).SetInt(scale),
		)
	} else if value.Exp < 0 {
		scale := new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(int64(-value.Exp)),
			nil,
		)

		rat.Quo(
			rat,
			new(big.Rat).SetInt(scale),
		)
	}

	result, _ := rat.Float64()

	return &result, nil
}

var _ domain.RestaurantRepository = (*PostgresRestaurantRepository)(nil)

// Keep time imported as part of the domain persistence contract if
// generated PostgreSQL timestamps are represented as time.Time.
var _ time.Time
