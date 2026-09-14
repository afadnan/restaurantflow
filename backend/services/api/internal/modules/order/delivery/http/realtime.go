package httpdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"restaurantflow/internal/platform/redis"
)

type RealtimeBridge struct {
	Subscriber *redis.Subscriber
	Hub        *Hub
	Logger     *slog.Logger
}

func NewRealtimeBridge(
	subscriber *redis.Subscriber,
	hub *Hub,
	logger *slog.Logger,
) *RealtimeBridge {
	if logger == nil {
		logger = slog.Default()
	}

	return &RealtimeBridge{
		Subscriber: subscriber,
		Hub:        hub,
		Logger:     logger,
	}
}

func (b *RealtimeBridge) RunTenant(
	ctx context.Context,
	tenantID uuid.UUID,
) error {
	return b.Subscriber.SubscribeTenant(
		ctx,
		tenantID,
		func(
			ctx context.Context,
			messageTenantID uuid.UUID,
			payload []byte,
		) error {
			if messageTenantID != tenantID {
				return fmt.Errorf("tenant mismatch in realtime bridge")
			}

			if !json.Valid(payload) {
				return fmt.Errorf("invalid event payload")
			}

			b.Hub.Broadcast(
				ctx,
				tenantID,
				payload,
			)

			return nil
		},
	)
}
