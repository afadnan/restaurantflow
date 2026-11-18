package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	menudomain "github.com/afadnan/restaurantflow/services/api/internal/modules/menu/domain"
	orderdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	db "github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

type fakeOutboxStore struct {
	events    []db.ClaimOutboxEventsRow
	published []db.MarkOutboxEventPublishedParams
	failed    []db.MarkOutboxEventFailedParams
	claimErr  error
}

func (f *fakeOutboxStore) ClaimOutboxEvents(
	_ context.Context,
	_ db.ClaimOutboxEventsParams,
) ([]db.ClaimOutboxEventsRow, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}

	return f.events, nil
}

func (f *fakeOutboxStore) MarkOutboxEventPublished(
	_ context.Context,
	params db.MarkOutboxEventPublishedParams,
) error {
	f.published = append(f.published, params)
	return nil
}

func (f *fakeOutboxStore) MarkOutboxEventFailed(
	_ context.Context,
	params db.MarkOutboxEventFailedParams,
) error {
	f.failed = append(f.failed, params)
	return nil
}

type fakePublisher struct {
	events []sharedevents.DomainEvent
	err    error
}

func (f *fakePublisher) Publish(
	_ context.Context,
	event sharedevents.DomainEvent,
) error {
	if f.err != nil {
		return f.err
	}

	f.events = append(f.events, event)
	return nil
}

func marshalEvent(event sharedevents.DomainEvent) ([]byte, error) {
	switch e := event.(type) {
	case orderdomain.OrderCreatedEvent:
		return json.Marshal(e)

	case orderdomain.KDSStateUpdatedEvent:
		return json.Marshal(e)

	default:
		return nil, errors.New("unsupported event type")
	}
}

func newOrderCreatedPayload(t *testing.T) []byte {
	t.Helper()

	event := orderdomain.OrderCreatedEvent{
		ID:         uuid.New(),
		Tenant:     uuid.New(),
		OrderID:    uuid.New(),
		CustomerID: uuid.New(),
		State:      orderdomain.OrderStatePending,
		TotalMinor: 12500,
		Currency:   "INR",
		Occurred:   time.Now().UTC(),
	}

	payload, err := marshalEvent(event)
	require.NoError(t, err)

	return payload
}

func TestDispatcher_DispatchesEvent(t *testing.T) {
	eventID := uuid.New()
	tenantID := uuid.New()

	store := &fakeOutboxStore{
		events: []db.ClaimOutboxEventsRow{
			{
				ID:        uuidToPgUUID(eventID),
				TenantID:  uuidToPgUUID(tenantID),
				EventType: string(orderdomain.EventOrderCreated),
				Payload:   newOrderCreatedPayload(t),
			},
		},
	}

	publisher := &fakePublisher{}

	dispatcher := NewDispatcher(
		store,
		publisher,
		nil,
		DispatcherConfig{},
	)

	err := dispatcher.dispatchBatch(context.Background())

	require.NoError(t, err)
	require.Len(t, publisher.events, 1)
	require.Len(t, store.published, 1)
	require.Empty(t, store.failed)

	require.True(
		t,
		bytes.Equal(
			eventID[:],
			store.published[0].ID.Bytes[:],
		),
	)
}

func TestDispatcher_MarksEventFailedWhenPublishFails(t *testing.T) {
	store := &fakeOutboxStore{
		events: []db.ClaimOutboxEventsRow{
			{
				ID:        uuidToPgUUID(uuid.New()),
				TenantID:  uuidToPgUUID(uuid.New()),
				EventType: string(orderdomain.EventOrderCreated),
				Payload:   newOrderCreatedPayload(t),
			},
		},
	}

	publisher := &fakePublisher{
		err: errors.New("redis unavailable"),
	}

	dispatcher := NewDispatcher(
		store,
		publisher,
		nil,
		DispatcherConfig{},
	)

	err := dispatcher.dispatchBatch(context.Background())

	require.NoError(t, err)
	require.Empty(t, store.published)
	require.Len(t, store.failed, 1)

	require.Equal(
		t,
		"redis unavailable",
		store.failed[0].LastError.String,
	)

	require.True(
		t,
		store.failed[0].LastError.Valid,
	)
}

func TestDecodeEvent_SupportsAllMenuEventTypes(t *testing.T) {
	tenantID := uuid.New()
	restaurantID := uuid.New()
	occurredAt := time.Now().UTC()

	tests := []struct {
		name  string
		event sharedevents.DomainEvent
	}{
		{
			name: "category created",
			event: menudomain.CategoryEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Category: uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeCategoryCreated,
			},
		},
		{
			name: "category updated",
			event: menudomain.CategoryEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Category: uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeCategoryUpdated,
			},
		},
		{
			name: "category deactivated",
			event: menudomain.CategoryEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Category: uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeCategoryDeactivated,
			},
		},
		{
			name: "item created",
			event: menudomain.ItemEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Item:     uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeItemCreated,
			},
		},
		{
			name: "item updated",
			event: menudomain.ItemEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Item:     uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeItemUpdated,
			},
		},
		{
			name: "item deactivated",
			event: menudomain.ItemEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Item:     uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeItemDeactivated,
			},
		},
		{
			name: "variant created",
			event: menudomain.VariantEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Variant:  uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeVariantCreated,
			},
		},
		{
			name: "variant updated",
			event: menudomain.VariantEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Variant:  uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeVariantUpdated,
			},
		},
		{
			name: "variant deactivated",
			event: menudomain.VariantEvent{
				ID:       uuid.New(),
				Tenant:   tenantID,
				Variant:  uuid.New(),
				Occurred: occurredAt,
				Type:     menudomain.EventTypeVariantDeactivated,
			},
		},
		{
			name: "addon created",
			event: menudomain.AddonEvent{
				ID:         uuid.New(),
				Tenant:     tenantID,
				Restaurant: restaurantID,
				Addon:      uuid.New(),
				Occurred:   occurredAt,
				Type:       menudomain.EventTypeAddonCreated,
			},
		},
		{
			name: "addon updated",
			event: menudomain.AddonEvent{
				ID:         uuid.New(),
				Tenant:     tenantID,
				Restaurant: restaurantID,
				Addon:      uuid.New(),
				Occurred:   occurredAt,
				Type:       menudomain.EventTypeAddonUpdated,
			},
		},
		{
			name: "addon deactivated",
			event: menudomain.AddonEvent{
				ID:         uuid.New(),
				Tenant:     tenantID,
				Restaurant: restaurantID,
				Addon:      uuid.New(),
				Occurred:   occurredAt,
				Type:       menudomain.EventTypeAddonDeactivated,
			},
		},
		{
			name: "item addon attached",
			event: menudomain.ItemAddonEvent{
				ID:          uuid.New(),
				Tenant:      tenantID,
				Restaurant:  restaurantID,
				Item:        uuid.New(),
				Addon:       uuid.New(),
				IsRequired:  true,
				MaxQuantity: 2,
				Occurred:    occurredAt,
				Type:        menudomain.EventTypeItemAddonAttached,
			},
		},
		{
			name: "item addon updated",
			event: menudomain.ItemAddonEvent{
				ID:          uuid.New(),
				Tenant:      tenantID,
				Restaurant:  restaurantID,
				Item:        uuid.New(),
				Addon:       uuid.New(),
				IsRequired:  false,
				MaxQuantity: 3,
				Occurred:    occurredAt,
				Type:        menudomain.EventTypeItemAddonUpdated,
			},
		},
		{
			name: "item addon detached",
			event: menudomain.ItemAddonEvent{
				ID:          uuid.New(),
				Tenant:      tenantID,
				Restaurant:  restaurantID,
				Item:        uuid.New(),
				Addon:       uuid.New(),
				IsRequired:  false,
				MaxQuantity: 1,
				Occurred:    occurredAt,
				Type:        menudomain.EventTypeItemAddonDetached,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(tt.event)
			require.NoError(t, err)

			decoded, err := decodeEvent(tt.event.EventType(), payload)
			require.NoError(t, err)
			require.Equal(t, tt.event.EventID(), decoded.EventID())
			require.Equal(t, tt.event.TenantID(), decoded.TenantID())
			require.Equal(t, tt.event.EventType(), decoded.EventType())
		})
	}
}

func TestDecodeEvent_RejectsMismatchedEventType(t *testing.T) {
	event := menudomain.ItemEvent{
		ID:       uuid.New(),
		Tenant:   uuid.New(),
		Item:     uuid.New(),
		Occurred: time.Now().UTC(),
		Type:     menudomain.EventTypeItemCreated,
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	_, err = decodeEvent(menudomain.EventTypeItemUpdated, payload)
	require.Error(t, err)
	require.Contains(t, err.Error(), "event type mismatch")
}

func TestDecodeEvent_RejectsUnsupportedEventType(t *testing.T) {
	_, err := decodeEvent(
		sharedevents.EventType("unknown.event"),
		[]byte(`{}`),
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported outbox event type")
}
