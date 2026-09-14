package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventOrderCreated    EventType = "order.created"
	EventKDSStateUpdated EventType = "kds.state.updated"
)

type DomainEvent interface {
	EventType() EventType
	Tenant() uuid.UUID
	AggregateID() uuid.UUID
	OccurredAt() time.Time
}

type OrderCreatedEvent struct {
	EventID     uuid.UUID  `json:"event_id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	OrderID     uuid.UUID  `json:"order_id"`
	CustomerID  uuid.UUID  `json:"customer_id"`
	State       OrderState `json:"state"`
	TotalMinor  int64      `json:"total_minor"`
	Currency    string     `json:"currency"`
	OccurredAtT time.Time  `json:"occurred_at"`
}

func (e OrderCreatedEvent) EventType() EventType {
	return EventOrderCreated
}

func (e OrderCreatedEvent) Tenant() uuid.UUID {
	return e.TenantID
}

func (e OrderCreatedEvent) AggregateID() uuid.UUID {
	return e.OrderID
}

func (e OrderCreatedEvent) OccurredAt() time.Time {
	return e.OccurredAtT
}

type KDSStateUpdatedEvent struct {
	EventID       uuid.UUID `json:"event_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	OrderID       uuid.UUID `json:"order_id"`
	PreviousState KDSState  `json:"previous_state"`
	State         KDSState  `json:"state"`
	OccurredAtT   time.Time `json:"occurred_at"`
}

func (e KDSStateUpdatedEvent) EventType() EventType {
	return EventKDSStateUpdated
}

func (e KDSStateUpdatedEvent) Tenant() uuid.UUID {
	return e.TenantID
}

func (e KDSStateUpdatedEvent) AggregateID() uuid.UUID {
	return e.OrderID
}

func (e KDSStateUpdatedEvent) OccurredAt() time.Time {
	return e.OccurredAtT
}
