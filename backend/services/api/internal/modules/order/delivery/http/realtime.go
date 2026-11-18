package httpdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	orderdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/redis"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

const (
	defaultRetryInitialDelay = time.Second
	defaultRetryMaxDelay     = 30 * time.Second
)

type patternSubscriber interface {
	SubscribePattern(
		ctx context.Context,
		handler redis.PatternMessageHandler,
	) error
}

type RealtimeBridge struct {
	Subscriber patternSubscriber
	Hub        *Hub
	Logger     *slog.Logger

	retryInitialDelay time.Duration
	retryMaxDelay     time.Duration
}

func NewRealtimeBridge(
	subscriber patternSubscriber,
	hub *Hub,
	logger *slog.Logger,
) *RealtimeBridge {
	if logger == nil {
		logger = slog.Default()
	}

	return &RealtimeBridge{
		Subscriber:        subscriber,
		Hub:               hub,
		Logger:            logger,
		retryInitialDelay: defaultRetryInitialDelay,
		retryMaxDelay:     defaultRetryMaxDelay,
	}
}

// Run subscribes to tenant event channels and forwards only KDS state updates.
// Unexpected subscription termination triggers exponential-backoff retries.
func (b *RealtimeBridge) Run(ctx context.Context) error {
	if b == nil {
		return fmt.Errorf("realtime bridge is required")
	}

	if b.Subscriber == nil {
		return fmt.Errorf("realtime bridge subscriber is required")
	}

	if b.Hub == nil {
		return fmt.Errorf("realtime bridge hub is required")
	}

	if ctx == nil {
		return fmt.Errorf("realtime bridge context is required")
	}

	initialDelay := b.retryInitialDelay
	if initialDelay <= 0 {
		initialDelay = defaultRetryInitialDelay
	}

	maxDelay := b.retryMaxDelay
	if maxDelay < initialDelay {
		maxDelay = defaultRetryMaxDelay
		if maxDelay < initialDelay {
			maxDelay = initialDelay
		}
	}

	delay := initialDelay

	handler := func(
		ctx context.Context,
		tenantID uuid.UUID,
		eventType sharedevents.EventType,
		payload []byte,
	) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		if tenantID == uuid.Nil {
			return fmt.Errorf("realtime event tenant id is required")
		}

		if eventType != orderdomain.EventKDSStateUpdated {
			return nil
		}

		if !json.Valid(payload) {
			return fmt.Errorf("invalid KDS event payload")
		}

		var event struct {
			EventID  uuid.UUID `json:"event_id"`
			TenantID uuid.UUID `json:"tenant_id"`
		}

		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode KDS event payload: %w", err)
		}

		if event.EventID == uuid.Nil {
			return fmt.Errorf("KDS event id is required")
		}

		if event.TenantID == uuid.Nil || event.TenantID != tenantID {
			return fmt.Errorf(
				"KDS event tenant mismatch: channel tenant %s, payload tenant %s",
				tenantID,
				event.TenantID,
			)
		}

		b.Hub.Broadcast(ctx, tenantID, payload)

		return nil
	}

	for {
		if ctx.Err() != nil {
			return nil
		}

		err := b.Subscriber.SubscribePattern(ctx, handler)

		// A normal subscription should remain active until its context
		// is cancelled. Returning nil unexpectedly is also a disconnect.
		if ctx.Err() != nil {
			return nil
		}

		if err == nil {
			err = fmt.Errorf("Redis pattern subscription ended unexpectedly")
		}

		b.Logger.WarnContext(
			ctx,
			"KDS Redis subscription stopped; retrying",
			"retry_in", delay,
			"error", err,
		)

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil

		case <-timer.C:
		}

		if delay < maxDelay {
			if delay > maxDelay/2 {
				delay = maxDelay
			} else {
				delay *= 2
			}
		}
	}
}
