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
	EventTypeItemCreated         sharedevents.EventType = "menu.item.created"
	EventTypeItemUpdated         sharedevents.EventType = "menu.item.updated"
	EventTypeItemDeactivated     sharedevents.EventType = "menu.item.deactivated"
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

var (
	_ sharedevents.DomainEvent = CategoryEvent{}
	_ sharedevents.DomainEvent = ItemEvent{}
)
