package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	orderdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
	redisplatform "github.com/afadnan/restaurantflow/services/api/internal/platform/redis"
)

type fakePatternSubscriber struct {
	mu      sync.Mutex
	handler redisplatform.PatternMessageHandler
	calls   int
	results []error
	entered chan int
}

func (f *fakePatternSubscriber) SubscribePattern(
	ctx context.Context,
	handler redisplatform.PatternMessageHandler,
) error {
	f.mu.Lock()
	f.handler = handler
	f.calls++
	call := f.calls

	var result error
	hasResult := false

	if len(f.results) > 0 {
		result = f.results[0]
		f.results = f.results[1:]
		hasResult = true
	}
	f.mu.Unlock()

	if f.entered != nil {
		select {
		case f.entered <- call:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if hasResult {
		return result
	}

	<-ctx.Done()
	return ctx.Err()
}

func (f *fakePatternSubscriber) getHandler() redisplatform.PatternMessageHandler {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.handler
}

func (f *fakePatternSubscriber) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func newTestBridge(t *testing.T) (*RealtimeBridge, *Hub, *fakePatternSubscriber) {
	t.Helper()

	subscriber := &fakePatternSubscriber{}
	hub := NewHub(nil)
	bridge := NewRealtimeBridge(subscriber, hub, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- bridge.Run(ctx)
	}()

	select {
	case <-subscriber.entered:
		// Not used by the default fake; retained for consistency.
	case <-time.After(20 * time.Millisecond):
	}

	if subscriber.getHandler() == nil {
		cancel()
		<-done
		t.Fatal("bridge did not register subscriber handler")
	}

	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("bridge did not stop after context cancellation")
		}
	})

	return bridge, hub, subscriber
}

func TestRealtimeBridgeBroadcastsKDSOnlyToMatchingTenant(t *testing.T) {
	_, hub, subscriber := newTestBridge(t)

	tenantA := uuid.New()
	tenantB := uuid.New()

	clientA := &Client{tenantID: tenantA, send: make(chan []byte, 1)}
	clientB := &Client{tenantID: tenantB, send: make(chan []byte, 1)}

	hub.Register(clientA)
	hub.Register(clientB)
	defer hub.Unregister(clientA)
	defer hub.Unregister(clientB)

	eventPayload, err := json.Marshal(struct {
		EventID  uuid.UUID `json:"event_id"`
		TenantID uuid.UUID `json:"tenant_id"`
		OrderID  uuid.UUID `json:"order_id"`
	}{
		EventID:  uuid.New(),
		TenantID: tenantA,
		OrderID:  uuid.New(),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if err := subscriber.getHandler()(
		context.Background(),
		tenantA,
		orderdomain.EventKDSStateUpdated,
		eventPayload,
	); err != nil {
		t.Fatalf("handle KDS event: %v", err)
	}

	select {
	case got := <-clientA.send:
		if string(got) != string(eventPayload) {
			t.Fatalf("unexpected payload: got %s, want %s", got, eventPayload)
		}
	default:
		t.Fatal("tenant A client did not receive KDS event")
	}

	select {
	case got := <-clientB.send:
		t.Fatalf("tenant B received tenant A event: %s", got)
	default:
	}
}

func TestRealtimeBridgeIgnoresNonKDSEvents(t *testing.T) {
	_, hub, subscriber := newTestBridge(t)

	tenantID := uuid.New()
	client := &Client{tenantID: tenantID, send: make(chan []byte, 1)}

	hub.Register(client)
	defer hub.Unregister(client)

	err := subscriber.getHandler()(
		context.Background(),
		tenantID,
		orderdomain.EventOrderCreated,
		[]byte(`{"not":"a KDS event"}`),
	)
	if err != nil {
		t.Fatalf("non-KDS event should be ignored: %v", err)
	}

	select {
	case got := <-client.send:
		t.Fatalf("client received non-KDS event: %s", got)
	default:
	}
}

func TestRealtimeBridgeRejectsTenantMismatch(t *testing.T) {
	_, _, subscriber := newTestBridge(t)

	channelTenant := uuid.New()
	payloadTenant := uuid.New()

	payload, err := json.Marshal(struct {
		EventID  uuid.UUID `json:"event_id"`
		TenantID uuid.UUID `json:"tenant_id"`
	}{
		EventID:  uuid.New(),
		TenantID: payloadTenant,
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	err = subscriber.getHandler()(
		context.Background(),
		channelTenant,
		orderdomain.EventKDSStateUpdated,
		payload,
	)
	if err == nil {
		t.Fatal("expected tenant mismatch to be rejected")
	}
}

func TestRealtimeBridgeRejectsInvalidKDSPayloads(t *testing.T) {
	_, _, subscriber := newTestBridge(t)

	tenantID := uuid.New()

	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "invalid JSON", payload: []byte(`{"event_id":`)},
		{
			name: "missing event ID",
			payload: []byte(`{"tenant_id":"` +
				tenantID.String() + `"}`),
		},
		{
			name: "missing tenant ID",
			payload: []byte(`{"event_id":"` +
				uuid.NewString() + `"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := subscriber.getHandler()(
				context.Background(),
				tenantID,
				orderdomain.EventKDSStateUpdated,
				tt.payload,
			)
			if err == nil {
				t.Fatal("expected invalid KDS payload to be rejected")
			}
		})
	}
}

func TestRealtimeBridgeRejectsNilDependencies(t *testing.T) {
	tests := []struct {
		name   string
		bridge *RealtimeBridge
	}{
		{name: "nil bridge"},
		{
			name:   "nil subscriber",
			bridge: &RealtimeBridge{Hub: NewHub(nil)},
		},
		{
			name: "nil hub",
			bridge: &RealtimeBridge{
				Subscriber: &fakePatternSubscriber{},
			},
		},
		{
			name: "nil context",
			bridge: &RealtimeBridge{
				Subscriber: &fakePatternSubscriber{},
				Hub:        NewHub(nil),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.name == "nil context" {
				err = tt.bridge.Run(nil)
			} else {
				err = tt.bridge.Run(context.Background())
			}
			if err == nil {
				t.Fatal("expected missing dependency to be rejected")
			}
		})
	}
}

func TestRealtimeBridgeRetriesAfterSubscriptionFailure(t *testing.T) {
	subscriber := &fakePatternSubscriber{
		results: []error{
			errors.New("redis unavailable"),
		},
		entered: make(chan int, 3),
	}

	bridge := NewRealtimeBridge(subscriber, NewHub(nil), slog.New(slog.NewTextHandler(io.Discard, nil)))
	bridge.retryInitialDelay = 10 * time.Millisecond
	bridge.retryMaxDelay = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- bridge.Run(ctx)
	}()

	select {
	case call := <-subscriber.entered:
		if call != 1 {
			t.Fatalf("first subscription call = %d, want 1", call)
		}
	case <-time.After(time.Second):
		t.Fatal("first subscription attempt did not happen")
	}

	select {
	case call := <-subscriber.entered:
		if call != 2 {
			t.Fatalf("second subscription call = %d, want 2", call)
		}
	case <-time.After(time.Second):
		t.Fatal("bridge did not retry after subscription failure")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bridge returned error on cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bridge did not stop after cancellation")
	}
}

func TestRealtimeBridgeStopsImmediatelyWhenContextCancelled(t *testing.T) {
	subscriber := &fakePatternSubscriber{
		entered: make(chan int, 1),
	}

	bridge := NewRealtimeBridge(subscriber, NewHub(nil), nil)
	bridge.retryInitialDelay = time.Second

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- bridge.Run(ctx)
	}()

	select {
	case <-subscriber.entered:
	case <-time.After(time.Second):
		t.Fatal("subscription did not start")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bridge returned error on cancellation: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("bridge did not stop immediately after cancellation")
	}
}

func TestRealtimeBridgeRetriesAfterUnexpectedCleanExit(t *testing.T) {
	subscriber := &fakePatternSubscriber{
		results: []error{nil},
		entered: make(chan int, 3),
	}

	bridge := NewRealtimeBridge(subscriber, NewHub(nil), nil)
	bridge.retryInitialDelay = 10 * time.Millisecond
	bridge.retryMaxDelay = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- bridge.Run(ctx)
	}()

	for expected := 1; expected <= 2; expected++ {
		select {
		case call := <-subscriber.entered:
			if call != expected {
				t.Fatalf("subscription call = %d, want %d", call, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscription attempt %d did not happen", expected)
		}
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bridge returned error on cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bridge did not stop")
	}
}
