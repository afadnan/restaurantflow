package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	db "github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
	redisplatform "github.com/afadnan/restaurantflow/services/api/internal/platform/redis"
)

const (
	defaultBatchSize    int32         = 50
	defaultLease        time.Duration = 30 * time.Second
	defaultPollInterval time.Duration = time.Second
)

// OutboxStore contains only the database operations required by the
// dispatcher. The generated sqlc Queries type implements this interface.
type OutboxStore interface {
	ClaimOutboxEvents(
		context.Context,
		db.ClaimOutboxEventsParams,
	) ([]db.ClaimOutboxEventsRow, error)

	MarkOutboxEventPublished(
		context.Context,
		db.MarkOutboxEventPublishedParams,
	) error

	MarkOutboxEventFailed(
		context.Context,
		db.MarkOutboxEventFailedParams,
	) error
}

// EventPublisher is the transport used after an outbox event has been
// durably stored in PostgreSQL.
type EventPublisher interface {
	Publish(context.Context, domain.DomainEvent) error
}

type Dispatcher struct {
	store        OutboxStore
	publisher    EventPublisher
	logger       *slog.Logger
	batchSize    int32
	lease        time.Duration
	pollInterval time.Duration
}

type DispatcherConfig struct {
	BatchSize    int32
	Lease        time.Duration
	PollInterval time.Duration
}

func NewDispatcher(
	store OutboxStore,
	publisher EventPublisher,
	logger *slog.Logger,
	cfg DispatcherConfig,
) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}

	if cfg.Lease <= 0 {
		cfg.Lease = defaultLease
	}

	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}

	return &Dispatcher{
		store:        store,
		publisher:    publisher,
		logger:       logger,
		batchSize:    cfg.BatchSize,
		lease:        cfg.Lease,
		pollInterval: cfg.PollInterval,
	}
}

func (d *Dispatcher) Run(ctx context.Context) error {
	if d.store == nil {
		return errors.New("dispatcher outbox store is nil")
	}

	if d.publisher == nil {
		return errors.New("dispatcher publisher is nil")
	}

	d.logger.InfoContext(
		ctx,
		"outbox dispatcher started",
		"batch_size", d.batchSize,
		"lease", d.lease,
		"poll_interval", d.pollInterval,
	)

	if err := d.dispatchBatch(ctx); err != nil &&
		!errors.Is(err, context.Canceled) {
		d.logger.ErrorContext(
			ctx,
			"initial outbox dispatch failed",
			"error", err,
		)
	}

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info(
				"outbox dispatcher stopping",
				"reason",
				ctx.Err(),
			)

			return ctx.Err()

		case <-ticker.C:
			if err := d.dispatchBatch(ctx); err != nil {
				if errors.Is(err, context.Canceled) ||
					errors.Is(err, context.DeadlineExceeded) {
					return err
				}

				d.logger.ErrorContext(
					ctx,
					"outbox dispatch batch failed",
					"error",
					err,
				)
			}
		}
	}
}

func (d *Dispatcher) dispatchBatch(ctx context.Context) error {
	claimToken := uuid.New()

	rows, err := d.store.ClaimOutboxEvents(
		ctx,
		db.ClaimOutboxEventsParams{
			ClaimToken: pgtype.UUID{
				Bytes: claimToken,
				Valid: true,
			},
			LeaseSeconds: int32(d.lease / time.Second),
			BatchSize:    d.batchSize,
		},
	)
	if err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}

	if len(rows) == 0 {
		return nil
	}

	d.logger.DebugContext(
		ctx,
		"claimed outbox events",
		"count", len(rows),
		"claim_token", claimToken,
	)

	for _, row := range rows {
		if err := d.dispatchEvent(ctx, row, claimToken); err != nil {
			d.logger.ErrorContext(
				ctx,
				"outbox event dispatch failed",
				"event_id", row.ID,
				"event_type", row.EventType,
				"tenant_id", row.TenantID,
				"error", err,
			)
		}
	}

	return nil
}

func (d *Dispatcher) dispatchEvent(
	ctx context.Context,
	row db.ClaimOutboxEventsRow,
	claimToken uuid.UUID,
) error {
	event, err := decodeEvent(
		domain.EventType(row.EventType),
		row.Payload,
	)
	if err != nil {
		if markErr := d.markFailed(
			ctx,
			row.ID,
			claimToken,
			err,
		); markErr != nil {
			return fmt.Errorf(
				"decode event: %v; mark failure: %w",
				err,
				markErr,
			)
		}

		return fmt.Errorf("decode event: %w", err)
	}

	if err := d.publisher.Publish(ctx, event); err != nil {
		if markErr := d.markFailed(
			ctx,
			row.ID,
			claimToken,
			err,
		); markErr != nil {
			return fmt.Errorf(
				"publish event: %v; mark failure: %w",
				err,
				markErr,
			)
		}

		return fmt.Errorf("publish event: %w", err)
	}

	if err := d.store.MarkOutboxEventPublished(
		ctx,
		db.MarkOutboxEventPublishedParams{
			ID:         row.ID,
			ClaimToken: uuidToPgUUID(claimToken),
		},
	); err != nil {
		return fmt.Errorf("mark event published: %w", err)
	}

	d.logger.DebugContext(
		ctx,
		"outbox event published",
		"event_id", row.ID,
		"event_type", row.EventType,
		"tenant_id", row.TenantID,
	)

	return nil
}

func (d *Dispatcher) markFailed(
	ctx context.Context,
	eventID pgtype.UUID,
	claimToken uuid.UUID,
	cause error,
) error {
	return d.store.MarkOutboxEventFailed(
		ctx,
		db.MarkOutboxEventFailedParams{
			ID:         eventID,
			ClaimToken: uuidToPgUUID(claimToken),
			LastError: pgtype.Text{
				String: cause.Error(),
				Valid:  true,
			},
		},
	)
}

func decodeEvent(
	eventType domain.EventType,
	payload []byte,
) (domain.DomainEvent, error) {
	switch eventType {
	case domain.EventOrderCreated:
		var event domain.OrderCreatedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode order.created payload: %w",
				err,
			)
		}

		if event.EventID == uuid.Nil {
			return nil, errors.New(
				"order.created event id is nil",
			)
		}

		return event, nil

	case domain.EventKDSStateUpdated:
		var event domain.KDSStateUpdatedEvent

		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf(
				"decode kds.state.updated payload: %w",
				err,
			)
		}

		if event.EventID == uuid.Nil {
			return nil, errors.New(
				"kds.state.updated event id is nil",
			)
		}

		return event, nil

	default:
		return nil, fmt.Errorf(
			"unsupported outbox event type: %q",
			eventType,
		)
	}
}

// Compile-time verification that the current Redis publisher implements
// the dispatcher publisher contract.
var _ EventPublisher = (*redisplatform.Publisher)(nil)
