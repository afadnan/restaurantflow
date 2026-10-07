package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeRestaurantCreated     EventType = "restaurant.created"
	EventTypeRestaurantUpdated     EventType = "restaurant.updated"
	EventTypeRestaurantActivated   EventType = "restaurant.activated"
	EventTypeRestaurantDeactivated EventType = "restaurant.deactivated"
	EventTypeRestaurantSuspended   EventType = "restaurant.suspended"
)

type DomainEvent interface {
	EventID() uuid.UUID
	TenantID() uuid.UUID
	AggregateID() uuid.UUID
	EventType() EventType
	OccurredAt() time.Time
}

type RestaurantCreatedEvent struct {
	ID         uuid.UUID
	Tenant     uuid.UUID
	Restaurant uuid.UUID
	Occurred   time.Time
}

func (e RestaurantCreatedEvent) EventID() uuid.UUID {
	return e.ID
}

func (e RestaurantCreatedEvent) TenantID() uuid.UUID {
	return e.Tenant
}

func (e RestaurantCreatedEvent) AggregateID() uuid.UUID {
	return e.Restaurant
}

func (e RestaurantCreatedEvent) EventType() EventType {
	return EventTypeRestaurantCreated
}

func (e RestaurantCreatedEvent) OccurredAt() time.Time {
	return e.Occurred
}

type RestaurantUpdatedEvent struct {
	ID         uuid.UUID
	Tenant     uuid.UUID
	Restaurant uuid.UUID
	Occurred   time.Time
}

func (e RestaurantUpdatedEvent) EventID() uuid.UUID {
	return e.ID
}

func (e RestaurantUpdatedEvent) TenantID() uuid.UUID {
	return e.Tenant
}

func (e RestaurantUpdatedEvent) AggregateID() uuid.UUID {
	return e.Restaurant
}

func (e RestaurantUpdatedEvent) EventType() EventType {
	return EventTypeRestaurantUpdated
}

func (e RestaurantUpdatedEvent) OccurredAt() time.Time {
	return e.Occurred
}

type RestaurantActivatedEvent struct {
	ID         uuid.UUID
	Tenant     uuid.UUID
	Restaurant uuid.UUID
	Occurred   time.Time
}

func (e RestaurantActivatedEvent) EventID() uuid.UUID {
	return e.ID
}

func (e RestaurantActivatedEvent) TenantID() uuid.UUID {
	return e.Tenant
}

func (e RestaurantActivatedEvent) AggregateID() uuid.UUID {
	return e.Restaurant
}

func (e RestaurantActivatedEvent) EventType() EventType {
	return EventTypeRestaurantActivated
}

func (e RestaurantActivatedEvent) OccurredAt() time.Time {
	return e.Occurred
}

type RestaurantDeactivatedEvent struct {
	ID         uuid.UUID
	Tenant     uuid.UUID
	Restaurant uuid.UUID
	Occurred   time.Time
}

func (e RestaurantDeactivatedEvent) EventID() uuid.UUID {
	return e.ID
}

func (e RestaurantDeactivatedEvent) TenantID() uuid.UUID {
	return e.Tenant
}

func (e RestaurantDeactivatedEvent) AggregateID() uuid.UUID {
	return e.Restaurant
}

func (e RestaurantDeactivatedEvent) EventType() EventType {
	return EventTypeRestaurantDeactivated
}

func (e RestaurantDeactivatedEvent) OccurredAt() time.Time {
	return e.Occurred
}

type RestaurantSuspendedEvent struct {
	ID         uuid.UUID
	Tenant     uuid.UUID
	Restaurant uuid.UUID
	Occurred   time.Time
}

func (e RestaurantSuspendedEvent) EventID() uuid.UUID {
	return e.ID
}

func (e RestaurantSuspendedEvent) TenantID() uuid.UUID {
	return e.Tenant
}

func (e RestaurantSuspendedEvent) AggregateID() uuid.UUID {
	return e.Restaurant
}

func (e RestaurantSuspendedEvent) EventType() EventType {
	return EventTypeRestaurantSuspended
}

func (e RestaurantSuspendedEvent) OccurredAt() time.Time {
	return e.Occurred
}
