package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

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
