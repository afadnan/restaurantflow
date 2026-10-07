package domain

import (
	"time"

	"github.com/google/uuid"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

const (
	EventOrderCreated    sharedevents.EventType = "order.created"
	EventKDSStateUpdated sharedevents.EventType = "kds.state.updated"
)

type OrderCreatedEvent struct {
	ID         uuid.UUID  `json:"event_id"`
	Tenant     uuid.UUID  `json:"tenant_id"`
	OrderID    uuid.UUID  `json:"order_id"`
	CustomerID uuid.UUID  `json:"customer_id"`
	State      OrderState `json:"state"`
	TotalMinor int64      `json:"total_minor"`
	Currency   string     `json:"currency"`
	Occurred   time.Time  `json:"occurred_at"`
}

func (e OrderCreatedEvent) EventID() uuid.UUID {
	return e.ID
}

func (e OrderCreatedEvent) TenantID() uuid.UUID {
	return e.Tenant
}

func (e OrderCreatedEvent) AggregateID() uuid.UUID {
	return e.OrderID
}

func (e OrderCreatedEvent) EventType() sharedevents.EventType {
	return EventOrderCreated
}

func (e OrderCreatedEvent) OccurredAt() time.Time {
	return e.Occurred
}

type KDSStateUpdatedEvent struct {
	ID            uuid.UUID `json:"event_id"`
	Tenant        uuid.UUID `json:"tenant_id"`
	OrderID       uuid.UUID `json:"order_id"`
	PreviousState KDSState  `json:"previous_state"`
	State         KDSState  `json:"state"`
	Occurred      time.Time `json:"occurred_at"`
}

func (e KDSStateUpdatedEvent) EventID() uuid.UUID {
	return e.ID
}

func (e KDSStateUpdatedEvent) TenantID() uuid.UUID {
	return e.Tenant
}

func (e KDSStateUpdatedEvent) AggregateID() uuid.UUID {
	return e.OrderID
}

func (e KDSStateUpdatedEvent) EventType() sharedevents.EventType {
	return EventKDSStateUpdated
}

func (e KDSStateUpdatedEvent) OccurredAt() time.Time {
	return e.Occurred
}

var (
	_ sharedevents.DomainEvent = OrderCreatedEvent{}
	_ sharedevents.DomainEvent = KDSStateUpdatedEvent{}
)
