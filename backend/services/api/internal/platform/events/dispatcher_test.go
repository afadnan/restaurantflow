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
