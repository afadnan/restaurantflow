package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

func TestKDSWebSocketUnavailableHandler(t *testing.T) {
	tests := []struct {
		name    string
		handler *Handler
	}{
		{name: "nil handler"},
		{name: "nil hub", handler: &Handler{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/ws/v1/kds", nil)
			response := httptest.NewRecorder()

			tt.handler.KDSWebSocket(response, request)

			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected status %d, got %d",
					http.StatusServiceUnavailable, response.Code)
			}
		})
	}
}

func TestKDSWebSocketRequiresTenantContext(t *testing.T) {
	handler := NewHandler(NewHub(nil))

	request := httptest.NewRequest(http.MethodGet, "/ws/v1/kds", nil)
	response := httptest.NewRecorder()

	handler.KDSWebSocket(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, response.Code)
	}
}

func TestKDSWebSocketRejectsUnapprovedOrigin(t *testing.T) {
	handler := NewHandlerWithOrigins(
		NewHub(nil),
		[]string{"app.example.com"},
	)
	tenantID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		ctx := middleware.WithTenant(r.Context(), tenantID)
		handler.KDSWebSocket(w, r.WithContext(ctx))
	}))
	defer server.Close()

	headers := make(http.Header)
	headers.Set("Origin", "https://evil.example.com")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, response, err := websocket.Dial(
		ctx,
		"ws"+server.URL[len("http"):],
		&websocket.DialOptions{HTTPHeader: headers},
	)
	if err == nil {
		t.Fatal("expected WebSocket handshake to reject unapproved origin")
	}
	if response == nil {
		t.Fatal("expected HTTP response for rejected WebSocket handshake")
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d",
			http.StatusForbidden, response.StatusCode)
	}
}

func TestKDSWebSocketRegistersAndUnregistersTenantClient(t *testing.T) {
	hub := NewHub(nil)
	handler := NewHandler(hub)
	tenantID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		ctx := middleware.WithTenant(r.Context(), tenantID)
		handler.KDSWebSocket(w, r.WithContext(ctx))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	serverURL.Scheme = "ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, serverURL.String(), nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}

	hub.mu.RLock()
	registered := len(hub.clients[tenantID])
	hub.mu.RUnlock()

	if registered != 1 {
		t.Fatalf("expected one client registered for tenant, got %d", registered)
	}

	if err := conn.Close(websocket.StatusNormalClosure, "test complete"); err != nil {
		t.Fatalf("close WebSocket client: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		hub.mu.RLock()
		remaining := len(hub.clients[tenantID])
		hub.mu.RUnlock()

		if remaining == 0 {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("WebSocket client was not unregistered after disconnect")
}
