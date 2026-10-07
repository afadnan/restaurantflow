package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type Publisher struct {
	client *redis.Client
	logger *slog.Logger
}

func NewPublisher(
	client *redis.Client,
	logger *slog.Logger,
) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}

	return &Publisher{
		client: client,
		logger: logger,
	}
}

func TenantChannel(tenantID string) string {
	return "restaurantflow:tenant:" + tenantID + ":orders"
}

type envelope struct {
	EventID     string                 `json:"event_id"`
	EventType   sharedevents.EventType `json:"event_type"`
	TenantID    string                 `json:"tenant_id"`
	AggregateID string                 `json:"aggregate_id"`
	OccurredAt  string                 `json:"occurred_at"`
	Payload     json.RawMessage        `json:"payload"`
}

func (p *Publisher) Publish(
	ctx context.Context,
	event sharedevents.DomainEvent,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if event == nil {
		return fmt.Errorf("domain event is required")
	}

	eventID := event.EventID()
	if eventID == [16]byte{} {
		return fmt.Errorf("event id is required")
	}

	tenantID := event.TenantID()
	if tenantID == [16]byte{} {
		return fmt.Errorf("event tenant id is required")
	}

	aggregateID := event.AggregateID()
	if aggregateID == [16]byte{} {
		return fmt.Errorf("event aggregate id is required")
	}

	eventType := event.EventType()
	if eventType == "" {
		return fmt.Errorf("event type is required")
	}

	occurredAt := event.OccurredAt()
	if occurredAt.IsZero() {
		return fmt.Errorf("event occurred at is required")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal domain event: %w", err)
	}

	envelopePayload := envelope{
		EventID:     eventID.String(),
		EventType:   eventType,
		TenantID:    tenantID.String(),
		AggregateID: aggregateID.String(),
		OccurredAt:  occurredAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		Payload:     payload,
	}

	message, err := json.Marshal(envelopePayload)
	if err != nil {
		return fmt.Errorf("marshal event envelope: %w", err)
	}

	channel := TenantChannel(tenantID.String())

	if err := p.client.Publish(
		ctx,
		channel,
		message,
	).Err(); err != nil {
		p.logger.ErrorContext(
			ctx,
			"redis event publish failed",
			"tenant_id", tenantID,
			"event_type", eventType,
			"aggregate_id", aggregateID,
			"event_id", eventID,
			"error", err,
		)

		return fmt.Errorf("publish redis event: %w", err)
	}

	return nil
}

func NewClient(
	addr string,
	password string,
	db int,
) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

var _ sharedevents.EventPublisher = (*Publisher)(nil)
