package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/middleware"
)

const (
	websocketClientBufferSize = 128
	websocketReadLimit        = 1024
)

type Client struct {
	conn     *websocket.Conn
	tenantID uuid.UUID
	send     chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]struct{}
	logger  *slog.Logger
	closed  bool
}

func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}

	return &Hub{
		clients: make(map[uuid.UUID]map[*Client]struct{}),
		logger:  logger,
	}
}

func (h *Hub) Register(client *Client) bool {
	if h == nil || client == nil {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return false
	}

	clients, exists := h.clients[client.tenantID]
	if !exists {
		clients = make(map[*Client]struct{})
		h.clients[client.tenantID] = clients
	}

	clients[client] = struct{}{}

	return true
}

func (h *Hub) Unregister(client *Client) {
	if h == nil || client == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.clients[client.tenantID]
	if !exists {
		return
	}

	if _, registered := clients[client]; !registered {
		return
	}

	delete(clients, client)
	close(client.send)

	if len(clients) == 0 {
		delete(h.clients, client.tenantID)
	}
}

// CloseAll prevents new registrations, removes all clients,
// closes their send channels, and forcefully closes their connections.
//
// It is safe to call multiple times.
func (h *Hub) CloseAll() {
	if h == nil {
		return
	}

	h.mu.Lock()

	if h.closed {
		h.mu.Unlock()
		return
	}

	h.closed = true

	clients := make([]*Client, 0)

	for tenantID, tenantClients := range h.clients {
		for client := range tenantClients {
			clients = append(clients, client)
			close(client.send)
		}

		delete(h.clients, tenantID)
	}

	h.mu.Unlock()

	for _, client := range clients {
		if client.conn == nil {
			continue
		}

		if err := client.conn.CloseNow(); err != nil {
			h.logger.Debug(
				"websocket force close failed",
				"tenant_id", client.tenantID,
				"error", err,
			)
		}
	}
}

func (h *Hub) Broadcast(
	ctx context.Context,
	tenantID uuid.UUID,
	payload []byte,
) {
	if h == nil || ctx == nil || ctx.Err() != nil {
		return
	}

	// Keep the read lock while sending. Unregister and CloseAll
	// require the write lock, preventing sends to closed channels.
	//
	// Sends are non-blocking so a slow client cannot block the Hub.
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return
	}

	for client := range h.clients[tenantID] {
		select {
		case client.send <- payload:
		default:
			h.logger.WarnContext(
				ctx,
				"dropping websocket message because client buffer is full",
				"tenant_id", tenantID,
			)
		}
	}
}

type Handler struct {
	Hub            *Hub
	AllowedOrigins []string
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		Hub: hub,
	}
}

// NewHandlerWithOrigins creates a WebSocket handler with explicitly
// allowed cross-origin frontend hosts.
//
// An empty origins list preserves the library's default same-origin
// protection. Do not use wildcard origins.
func NewHandlerWithOrigins(
	hub *Hub,
	allowedOrigins []string,
) *Handler {
	return &Handler{
		Hub:            hub,
		AllowedOrigins: append([]string(nil), allowedOrigins...),
	}
}

func (h *Handler) KDSWebSocket(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.Hub == nil {
		http.Error(
			w,
			"websocket service unavailable",
			http.StatusServiceUnavailable,
		)
		return
	}

	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			"tenant context required",
			http.StatusUnauthorized,
		)
		return
	}

	acceptOptions := &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	}

	if len(h.AllowedOrigins) > 0 {
		acceptOptions.OriginPatterns = h.AllowedOrigins
	}

	conn, err := websocket.Accept(w, r, acceptOptions)
	if err != nil {
		return
	}

	conn.SetReadLimit(websocketReadLimit)

	client := &Client{
		conn:     conn,
		tenantID: tenantID,
		send:     make(chan []byte, websocketClientBufferSize),
	}

	if !h.Hub.Register(client) {
		_ = conn.Close(
			websocket.StatusGoingAway,
			"server shutting down",
		)
		return
	}

	defer h.Hub.Unregister(client)

	// The reader and writer share a cancellable context.
	// If either loop exits, the other loop is stopped too.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	writerDone := make(chan struct{})

	go func() {
		defer close(writerDone)

		h.writeLoop(ctx, client)
		cancel()

		// Ensure a blocked reader is released when the writer exits.
		_ = conn.CloseNow()
	}()

	h.readLoop(ctx, client)

	cancel()

	// Wait for the writer to exit before unregistering the client.
	<-writerDone

	_ = conn.Close(
		websocket.StatusNormalClosure,
		"",
	)
}

func (h *Handler) readLoop(
	ctx context.Context,
	client *Client,
) {
	for {
		_, _, err := client.conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				h.Hub.logger.DebugContext(
					ctx,
					"websocket read ended",
					"tenant_id", client.tenantID,
					"error", err,
				)
			}

			return
		}
	}
}

func (h *Handler) writeLoop(
	ctx context.Context,
	client *Client,
) {
	for {
		select {
		case <-ctx.Done():
			return

		case payload, ok := <-client.send:
			if !ok {
				return
			}

			if err := client.conn.Write(
				ctx,
				websocket.MessageText,
				payload,
			); err != nil {
				if !errors.Is(err, context.Canceled) {
					h.Hub.logger.DebugContext(
						ctx,
						"websocket write ended",
						"tenant_id", client.tenantID,
						"error", err,
					)
				}

				return
			}
		}
	}
}

type KDSMessage struct {
	Type      string    `json:"type"`
	TenantID  uuid.UUID `json:"tenant_id"`
	OrderID   uuid.UUID `json:"order_id"`
	State     string    `json:"state"`
	Timestamp string    `json:"timestamp"`
}

func EncodeKDSMessage(
	message KDSMessage,
) ([]byte, error) {
	return json.Marshal(message)
}
