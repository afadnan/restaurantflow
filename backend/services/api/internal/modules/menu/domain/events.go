package domain

import (
	"time"

	"github.com/google/uuid"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

const (
	EventTypeCategoryCreated     sharedevents.EventType = "menu.category.created"
	EventTypeCategoryUpdated     sharedevents.EventType = "menu.category.updated"
	EventTypeCategoryDeactivated sharedevents.EventType = "menu.category.deactivated"

	EventTypeItemCreated     sharedevents.EventType = "menu.item.created"
	EventTypeItemUpdated     sharedevents.EventType = "menu.item.updated"
	EventTypeItemDeactivated sharedevents.EventType = "menu.item.deactivated"

	EventTypeVariantCreated     sharedevents.EventType = "menu.variant.created"
	EventTypeVariantUpdated     sharedevents.EventType = "menu.variant.updated"
	EventTypeVariantDeactivated sharedevents.EventType = "menu.variant.deactivated"
)

type CategoryEvent struct {
	ID       uuid.UUID
	Tenant   uuid.UUID
	Category uuid.UUID
	Occurred time.Time
	Type     sharedevents.EventType
}

func (e CategoryEvent) EventID() uuid.UUID                { return e.ID }
func (e CategoryEvent) TenantID() uuid.UUID               { return e.Tenant }
func (e CategoryEvent) AggregateID() uuid.UUID            { return e.Category }
func (e CategoryEvent) EventType() sharedevents.EventType { return e.Type }
func (e CategoryEvent) OccurredAt() time.Time             { return e.Occurred }

type ItemEvent struct {
	ID       uuid.UUID
	Tenant   uuid.UUID
	Item     uuid.UUID
	Occurred time.Time
	Type     sharedevents.EventType
}

func (e ItemEvent) EventID() uuid.UUID                { return e.ID }
func (e ItemEvent) TenantID() uuid.UUID               { return e.Tenant }
func (e ItemEvent) AggregateID() uuid.UUID            { return e.Item }
func (e ItemEvent) EventType() sharedevents.EventType { return e.Type }
func (e ItemEvent) OccurredAt() time.Time             { return e.Occurred }

type VariantEvent struct {
	ID       uuid.UUID
	Tenant   uuid.UUID
	Variant  uuid.UUID
	Occurred time.Time
	Type     sharedevents.EventType
}

func (e VariantEvent) EventID() uuid.UUID                { return e.ID }
func (e VariantEvent) TenantID() uuid.UUID               { return e.Tenant }
func (e VariantEvent) AggregateID() uuid.UUID            { return e.Variant }
func (e VariantEvent) EventType() sharedevents.EventType { return e.Type }
func (e VariantEvent) OccurredAt() time.Time             { return e.Occurred }

var (
	_ sharedevents.DomainEvent = CategoryEvent{}
	_ sharedevents.DomainEvent = ItemEvent{}
	_ sharedevents.DomainEvent = VariantEvent{}
)
