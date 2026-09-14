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

	"restaurantflow/internal/middleware"
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

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.clients[client.tenantID]

	if !exists {
		clients = make(map[*Client]struct{})
		h.clients[client.tenantID] = clients
	}

	clients[client] = struct{}{}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, exists := h.clients[client.tenantID]

	if !exists {
		return
	}

	delete(clients, client)

	close(client.send)

	if len(clients) == 0 {
		delete(h.clients, client.tenantID)
	}
}

func (h *Hub) Broadcast(
	ctx context.Context,
	tenantID uuid.UUID,
	payload []byte,
) {
	if err := ctx.Err(); err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients[tenantID]))

	for client := range h.clients[tenantID] {
		clients = append(clients, client)
	}

	h.mu.RUnlock()

	for _, client := range clients {
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
	Hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		Hub: hub,
	}
}

func (h *Handler) KDSWebSocket(
	w http.ResponseWriter,
	r *http.Request,
) {
	tenantID, err := middleware.TenantFromContext(r.Context())
	if err != nil {
		http.Error(
			w,
			"tenant context required",
			http.StatusUnauthorized,
		)
		return
	}

	conn, err := websocket.Accept(
		w,
		r,
		&websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		},
	)
	if err != nil {
		return
	}

	client := &Client{
		conn:     conn,
		tenantID: tenantID,
		send:     make(chan []byte, 128),
	}

	h.Hub.Register(client)

	defer func() {
		h.Hub.Unregister(client)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := r.Context()

	go h.writeLoop(ctx, client)

	h.readLoop(ctx, client)
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
