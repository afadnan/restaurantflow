package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
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
	EventID     string           `json:"event_id"`
	EventType   domain.EventType `json:"event_type"`
	TenantID    string           `json:"tenant_id"`
	AggregateID string           `json:"aggregate_id"`
	OccurredAt  string           `json:"occurred_at"`
	Payload     json.RawMessage  `json:"payload"`
}

func (p *Publisher) Publish(
	ctx context.Context,
	event domain.DomainEvent,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal domain event: %w", err)
	}

	envelopePayload := envelope{
		EventID:     eventID(event),
		EventType:   event.EventType(),
		TenantID:    event.Tenant().String(),
		AggregateID: event.AggregateID().String(),
		OccurredAt:  event.OccurredAt().UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		Payload:     payload,
	}

	message, err := json.Marshal(envelopePayload)
	if err != nil {
		return fmt.Errorf("marshal event envelope: %w", err)
	}

	channel := TenantChannel(event.Tenant().String())

	if err := p.client.Publish(
		ctx,
		channel,
		message,
	).Err(); err != nil {
		p.logger.ErrorContext(
			ctx,
			"redis event publish failed",
			"tenant_id", event.Tenant(),
			"event_type", event.EventType(),
			"aggregate_id", event.AggregateID(),
			"error", err,
		)

		return fmt.Errorf("publish redis event: %w", err)
	}

	return nil
}

func eventID(event domain.DomainEvent) string {
	switch value := event.(type) {
	case domain.OrderCreatedEvent:
		return value.EventID.String()

	case domain.KDSStateUpdatedEvent:
		return value.EventID.String()

	default:
		return ""
	}
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
