package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventType identifies the kind of domain event.
type EventType string

// DomainEvent is the contract implemented by all domain events
// that can be persisted to the transactional outbox.
type DomainEvent interface {
	EventID() uuid.UUID
	TenantID() uuid.UUID
	AggregateID() uuid.UUID
	EventType() EventType
	OccurredAt() time.Time
}

// Transaction represents the minimal transaction contract required
// by the application and event infrastructure layers.
//
// Database-specific concerns such as pgx.Tx remain outside this
// shared contract and belong to the infrastructure layer.
type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// EventRepository persists domain events within the same transaction
// as the aggregate state change.
type EventRepository interface {
	Append(
		ctx context.Context,
		tx Transaction,
		event DomainEvent,
	) error
}

// EventPublisher publishes domain events to an external transport
// such as Redis Pub/Sub after they have been committed to the database.
type EventPublisher interface {
	Publish(
		ctx context.Context,
		event DomainEvent,
	) error
}
