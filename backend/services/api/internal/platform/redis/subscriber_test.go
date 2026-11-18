package redis

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	sharedevents "github.com/afadnan/restaurantflow/shared/events"
)

func TestDecodePatternMessage(t *testing.T) {
	tenantID := uuid.New()

	payload := json.RawMessage(`{"event_id":"event-1","tenant_id":"` +
		tenantID.String() + `"}`)

	envelope, err := json.Marshal(map[string]any{
		"tenant_id":  tenantID.String(),
		"event_type": "kds.state.updated",
		"payload":    payload,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	tests := []struct {
		name       string
		channel    string
		payload    string
		wantErr    bool
		wantEvent  sharedevents.EventType
		wantTenant uuid.UUID
	}{
		{
			name:       "valid tenant KDS event",
			channel:    TenantChannel(tenantID.String()),
			payload:    string(envelope),
			wantEvent:  sharedevents.EventType("kds.state.updated"),
			wantTenant: tenantID,
		},
		{
			name:    "invalid channel prefix",
			channel: "other:tenant:" + tenantID.String() + ":orders",
			payload: string(envelope),
			wantErr: true,
		},
		{
			name:    "invalid tenant UUID in channel",
			channel: "restaurantflow:tenant:not-a-uuid:orders",
			payload: string(envelope),
			wantErr: true,
		},
		{
			name:    "channel tenant differs from envelope tenant",
			channel: TenantChannel(uuid.NewString()),
			payload: string(envelope),
			wantErr: true,
		},
		{
			name:    "malformed envelope JSON",
			channel: TenantChannel(tenantID.String()),
			payload: `{"tenant_id":`,
			wantErr: true,
		},
		{
			name:    "missing event type",
			channel: TenantChannel(tenantID.String()),
			payload: `{"tenant_id":"` + tenantID.String() + `","payload":{}}`,
			wantErr: true,
		},
		{
			name:    "missing event payload",
			channel: TenantChannel(tenantID.String()),
			payload: `{"tenant_id":"` + tenantID.String() + `","event_type":"kds.state.updated"}`,
			wantErr: true,
		},
		{
			name:    "invalid event payload JSON",
			channel: TenantChannel(tenantID.String()),
			payload: `{"tenant_id":"` + tenantID.String() + `","event_type":"kds.state.updated","payload":`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := &redis.Message{
				Channel: tt.channel,
				Payload: tt.payload,
			}

			gotTenant, gotEvent, gotPayload, err := decodePatternMessage(message)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("decodePatternMessage() error = %v", err)
			}

			if gotTenant != tt.wantTenant {
				t.Fatalf("tenant = %s, want %s", gotTenant, tt.wantTenant)
			}

			if gotEvent != tt.wantEvent {
				t.Fatalf("event type = %q, want %q", gotEvent, tt.wantEvent)
			}

			if !json.Valid(gotPayload) {
				t.Fatalf("decoded payload is invalid JSON: %s", gotPayload)
			}
		})
	}
}

func TestDecodePatternMessageRejectsNilMessage(t *testing.T) {
	_, _, _, err := decodePatternMessage(nil)
	if err == nil {
		t.Fatal("expected nil message to be rejected")
	}
}
