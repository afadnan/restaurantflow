package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type PostgresKDSRepository struct{}

func NewPostgresKDSRepository() *PostgresKDSRepository {
	return &PostgresKDSRepository{}
}

func (r *PostgresKDSRepository) Create(
	ctx context.Context,
	tx domain.Transaction,
	kds *domain.KDSOrder,
) error {
	if kds == nil {
		return errors.New("kds order is required")
	}

	if !kds.State.Valid() {
		return fmt.Errorf("invalid KDS state: %s", kds.State)
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	q := db.New(pgTx.Raw())

	_, err = q.CreateKDSTicket(ctx, db.CreateKDSTicketParams{
		TenantID:     uuidToPgUUID(kds.TenantID.UUID()),
		RestaurantID: uuidToPgUUID(kds.RestaurantID.UUID()),
		OrderID:      uuidToPgUUID(kds.OrderID.UUID()),
		Priority:     0,
	})
	if err != nil {
		return fmt.Errorf("create KDS ticket: %w", err)
	}

	return nil
}

func (r *PostgresKDSRepository) GetByOrderID(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	orderID uuid.UUID,
) (*domain.KDSOrder, error) {
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant id is required")
	}

	if orderID == uuid.Nil {
		return nil, errors.New("order id is required")
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return nil, err
	}

	q := db.New(pgTx.Raw())

	row, err := q.GetKDSTicketByOrderID(ctx, db.GetKDSTicketByOrderIDParams{
		TenantID: uuidToPgUUID(tenantID),
		OrderID:  uuidToPgUUID(orderID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrKDSNotFound
		}

		return nil, fmt.Errorf(
			"get KDS ticket for order %s: %w",
			orderID,
			err,
		)
	}

	return mapKDSTicket(row)
}

func (r *PostgresKDSRepository) UpdateState(
	ctx context.Context,
	tx domain.Transaction,
	tenantID uuid.UUID,
	orderID uuid.UUID,
	from domain.KDSState,
	to domain.KDSState,
) error {
	if tenantID == uuid.Nil {
		return errors.New("tenant id is required")
	}

	if orderID == uuid.Nil {
		return errors.New("order id is required")
	}

	if !from.Valid() {
		return fmt.Errorf("invalid current KDS state: %s", from)
	}

	if !to.Valid() {
		return fmt.Errorf("invalid target KDS state: %s", to)
	}

	pgTx, err := postgresTx(tx)
	if err != nil {
		return err
	}

	q := db.New(pgTx.Raw())

	current, err := q.GetKDSTicketByOrderID(ctx, db.GetKDSTicketByOrderIDParams{
		TenantID: uuidToPgUUID(tenantID),
		OrderID:  uuidToPgUUID(orderID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrKDSNotFound
		}

		return fmt.Errorf("get KDS ticket before update: %w", err)
	}

	currentState := domain.KDSState(current.Status)

	if currentState != from {
		return fmt.Errorf(
			"KDS state conflict for order %s: expected %s, actual %s",
			orderID,
			from,
			currentState,
		)
	}

	if err := validateKDSTransition(from, to); err != nil {
		return err
	}

	_, err = q.UpdateKDSStatus(ctx, db.UpdateKDSStatusParams{
		TenantID: uuidToPgUUID(tenantID),
		ID:       current.ID,
		Status:   db.KdsStatus(to),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrKDSNotFound
		}

		return fmt.Errorf(
			"update KDS state for order %s: %w",
			orderID,
			err,
		)
	}

	return nil
}

func mapKDSTicket(row db.KdsTicket) (*domain.KDSOrder, error) {
	tenantUUID, err := pgUUIDToUUID(row.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant id: %w", err)
	}

	restaurantUUID, err := pgUUIDToUUID(row.RestaurantID)
	if err != nil {
		return nil, fmt.Errorf("restaurant id: %w", err)
	}

	orderUUID, err := pgUUIDToUUID(row.OrderID)
	if err != nil {
		return nil, fmt.Errorf("order id: %w", err)
	}

	id, err := pgUUIDToUUID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("KDS ticket id: %w", err)
	}

	tenantID, err := domain.NewTenantID(tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("tenant id: %w", err)
	}

	restaurantID, err := domain.NewRestaurantID(restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("restaurant id: %w", err)
	}

	orderID, err := domain.NewOrderIDFromUUID(orderUUID)
	if err != nil {
		return nil, fmt.Errorf("order id: %w", err)
	}

	state := domain.KDSState(row.Status)
	if !state.Valid() {
		return nil, fmt.Errorf(
			"invalid persisted KDS state %q",
			row.Status,
		)
	}

	kds := &domain.KDSOrder{
		ID:           id,
		TenantID:     tenantID,
		RestaurantID: restaurantID,
		OrderID:      orderID,
		State:        state,
	}

	if row.CreatedAt.Valid {
		kds.CreatedAt = row.CreatedAt.Time
	}

	if row.UpdatedAt.Valid {
		kds.UpdatedAt = row.UpdatedAt.Time
	}

	return kds, nil
}

func validateKDSTransition(
	from domain.KDSState,
	to domain.KDSState,
) error {
	switch from {
	case domain.KDSStatePending:
		if to == domain.KDSStatePreparing {
			return nil
		}

	case domain.KDSStatePreparing:
		if to == domain.KDSStateReady {
			return nil
		}

	case domain.KDSStateReady:
		if to == domain.KDSStateCompleted {
			return nil
		}

	case domain.KDSStateCompleted:
		return domain.ErrInvalidStateTransition
	}

	return domain.ErrInvalidStateTransition
}

var _ domain.KDSRepository = (*PostgresKDSRepository)(nil)
