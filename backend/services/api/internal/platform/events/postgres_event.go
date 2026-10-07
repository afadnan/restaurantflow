package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	db "github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type PostgresEventRepository struct{}

func NewPostgresEventRepository() *PostgresEventRepository {
	return &PostgresEventRepository{}
}

func (r *PostgresEventRepository) Append(
	ctx context.Context,
	tx sharedevents.Transaction,
	event sharedevents.DomainEvent,
) error {
	if event == nil {
		return errors.New("domain event is required")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	eventID := event.EventID()
	if eventID == uuid.Nil {
		return errors.New("event id is required")
	}

	tenantID := event.TenantID()
	if tenantID == uuid.Nil {
		return errors.New("event tenant id is required")
	}

	aggregateID := event.AggregateID()
	if aggregateID == uuid.Nil {
		return errors.New("event aggregate id is required")
	}

	eventType := event.EventType()
	if eventType == "" {
		return errors.New("event type is required")
	}

	occurredAt := event.OccurredAt()
	if occurredAt.IsZero() {
		return errors.New("event occurred at is required")
	}

	postgresTx, ok := tx.(*postgres.Tx)
	if !ok {
		return errors.New("unsupported transaction type")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal domain event: %w", err)
	}

	queries := db.New(postgresTx.Raw())

	if err := queries.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		ID:          uuidToPgUUID(eventID),
		TenantID:    uuidToPgUUID(tenantID),
		AggregateID: uuidToPgUUID(aggregateID),
		EventType:   string(eventType),
		Payload:     payload,
		OccurredAt:  pgtype.Timestamptz{Time: occurredAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}

	return nil
}

func uuidToPgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: value,
		Valid: value != uuid.Nil,
	}
}

var _ sharedevents.EventRepository = (*PostgresEventRepository)(nil)
