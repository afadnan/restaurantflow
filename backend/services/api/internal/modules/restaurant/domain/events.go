package domain

import (
	"time"

	"github.com/google/uuid"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

const (
	EventTypeRestaurantCreated     sharedevents.EventType = "restaurant.created"
	EventTypeRestaurantUpdated     sharedevents.EventType = "restaurant.updated"
	EventTypeRestaurantActivated   sharedevents.EventType = "restaurant.activated"
	EventTypeRestaurantDeactivated sharedevents.EventType = "restaurant.deactivated"
	EventTypeRestaurantSuspended   sharedevents.EventType = "restaurant.suspended"
)

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

func (e RestaurantCreatedEvent) EventType() sharedevents.EventType {
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

func (e RestaurantUpdatedEvent) EventType() sharedevents.EventType {
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

func (e RestaurantActivatedEvent) EventType() sharedevents.EventType {
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

func (e RestaurantDeactivatedEvent) EventType() sharedevents.EventType {
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

func (e RestaurantSuspendedEvent) EventType() sharedevents.EventType {
	return EventTypeRestaurantSuspended
}

func (e RestaurantSuspendedEvent) OccurredAt() time.Time {
	return e.Occurred
}

var (
	_ sharedevents.DomainEvent = RestaurantCreatedEvent{}
	_ sharedevents.DomainEvent = RestaurantUpdatedEvent{}
	_ sharedevents.DomainEvent = RestaurantActivatedEvent{}
	_ sharedevents.DomainEvent = RestaurantDeactivatedEvent{}
	_ sharedevents.DomainEvent = RestaurantSuspendedEvent{}
)
