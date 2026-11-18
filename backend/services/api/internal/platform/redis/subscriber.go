package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

const tenantChannelPattern = "restaurantflow:tenant:*:orders"

type Subscriber struct {
	client *redis.Client
	logger *slog.Logger
}

func NewSubscriber(
	client *redis.Client,
	logger *slog.Logger,
) *Subscriber {
	if logger == nil {
		logger = slog.Default()
	}

	return &Subscriber{
		client: client,
		logger: logger,
	}
}

type MessageHandler func(
	ctx context.Context,
	tenantID uuid.UUID,
	payload []byte,
) error

type PatternMessageHandler func(
	ctx context.Context,
	tenantID uuid.UUID,
	eventType sharedevents.EventType,
	payload []byte,
) error

func (s *Subscriber) SubscribeTenant(
	ctx context.Context,
	tenantID uuid.UUID,
	handler MessageHandler,
) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("tenant id cannot be nil")
	}

	channel := TenantChannel(tenantID.String())
	pubsub := s.client.Subscribe(ctx, channel)

	defer func() {
		_ = pubsub.Close()
	}()

	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("redis subscription handshake: %w", err)
	}

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case message, ok := <-ch:
			if !ok {
				return fmt.Errorf("redis subscription channel closed")
			}

			if err := s.handleMessage(
				ctx,
				tenantID,
				message,
				handler,
			); err != nil {
				s.logger.ErrorContext(
					ctx,
					"redis message handler failed",
					"tenant_id", tenantID,
					"error", err,
				)
			}
		}
	}
}

func (s *Subscriber) SubscribePattern(
	ctx context.Context,
	handler PatternMessageHandler,
) error {
	if handler == nil {
		return fmt.Errorf("redis pattern message handler is required")
	}

	pubsub := s.client.PSubscribe(ctx, tenantChannelPattern)

	defer func() {
		_ = pubsub.Close()
	}()

	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("redis pattern subscription handshake: %w", err)
	}

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case message, ok := <-ch:
			if !ok {
				return fmt.Errorf("redis pattern subscription channel closed")
			}

			tenantID, eventType, payload, err := decodePatternMessage(message)
			if err != nil {
				s.logger.ErrorContext(
					ctx,
					"invalid redis pattern message",
					"channel", message.Channel,
					"error", err,
				)
				continue
			}

			if err := handler(
				ctx,
				tenantID,
				eventType,
				payload,
			); err != nil {
				s.logger.ErrorContext(
					ctx,
					"redis pattern message handler failed",
					"tenant_id", tenantID,
					"event_type", eventType,
					"error", err,
				)
			}
		}
	}
}

func decodePatternMessage(
	message *redis.Message,
) (uuid.UUID, sharedevents.EventType, []byte, error) {
	if message == nil {
		return uuid.Nil, "", nil, fmt.Errorf("redis message is required")
	}

	const prefix = "restaurantflow:tenant:"
	const suffix = ":orders"

	channel := message.Channel

	if !strings.HasPrefix(channel, prefix) ||
		!strings.HasSuffix(channel, suffix) {
		return uuid.Nil, "", nil, fmt.Errorf(
			"invalid tenant channel: %q",
			channel,
		)
	}

	tenantValue := strings.TrimSuffix(
		strings.TrimPrefix(channel, prefix),
		suffix,
	)

	tenantID, err := uuid.Parse(tenantValue)
	if err != nil || tenantID == uuid.Nil {
		return uuid.Nil, "", nil, fmt.Errorf(
			"invalid tenant id in redis channel: %q",
			tenantValue,
		)
	}

	var envelope struct {
		TenantID  string          `json:"tenant_id"`
		EventType string          `json:"event_type"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(
		[]byte(message.Payload),
		&envelope,
	); err != nil {
		return uuid.Nil, "", nil, fmt.Errorf(
			"decode redis envelope: %w",
			err,
		)
	}

	envelopeTenantID, err := uuid.Parse(envelope.TenantID)
	if err != nil || envelopeTenantID != tenantID {
		return uuid.Nil, "", nil, fmt.Errorf(
			"redis tenant isolation violation: channel tenant %s does not match envelope tenant %q",
			tenantID,
			envelope.TenantID,
		)
	}

	if envelope.EventType == "" {
		return uuid.Nil, "", nil, fmt.Errorf("redis event type is required")
	}

	if len(envelope.Payload) == 0 ||
		!json.Valid(envelope.Payload) {
		return uuid.Nil, "", nil, fmt.Errorf("invalid redis event payload")
	}

	return tenantID, sharedevents.EventType(envelope.EventType), envelope.Payload, nil
}

func (s *Subscriber) handleMessage(
	ctx context.Context,
	tenantID uuid.UUID,
	message *redis.Message,
	handler MessageHandler,
) error {
	var envelope struct {
		TenantID string          `json:"tenant_id"`
		Payload  json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(
		[]byte(message.Payload),
		&envelope,
	); err != nil {
		return fmt.Errorf("decode redis envelope: %w", err)
	}

	if envelope.TenantID != tenantID.String() {
		return fmt.Errorf(
			"redis tenant isolation violation: expected %s got %s",
			tenantID,
			envelope.TenantID,
		)
	}

	if err := handler(
		ctx,
		tenantID,
		envelope.Payload,
	); err != nil {
		return fmt.Errorf("handle tenant event: %w", err)
	}

	return nil
}
