package config

import (
	"testing"
	"time"
)

func TestLoadJWTConfig(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		issuer     string
		ttl        string
		wantErr    bool
		wantIssuer string
		wantTTL    time.Duration
	}{
		{
			name:       "valid configuration",
			secret:     "0123456789abcdef0123456789abcdef",
			issuer:     "restaurantflow-api",
			ttl:        "15m",
			wantIssuer: "restaurantflow-api",
			wantTTL:    15 * time.Minute,
		},
		{
			name:    "missing secret",
			secret:  "",
			issuer:  "restaurantflow-api",
			ttl:     "15m",
			wantErr: true,
		},
		{
			name:    "short secret",
			secret:  "short-secret",
			issuer:  "restaurantflow-api",
			ttl:     "15m",
			wantErr: true,
		},
		{
			name:       "default issuer and TTL",
			secret:     "0123456789abcdef0123456789abcdef",
			issuer:     "",
			ttl:        "",
			wantIssuer: "restaurantflow-api",
			wantTTL:    15 * time.Minute,
		},
		{
			name:    "invalid duration",
			secret:  "0123456789abcdef0123456789abcdef",
			issuer:  "restaurantflow-api",
			ttl:     "invalid",
			wantErr: true,
		},
		{
			name:    "zero duration",
			secret:  "0123456789abcdef0123456789abcdef",
			issuer:  "restaurantflow-api",
			ttl:     "0s",
			wantErr: true,
		},
		{
			name:    "negative duration",
			secret:  "0123456789abcdef0123456789abcdef",
			issuer:  "restaurantflow-api",
			ttl:     "-15m",
			wantErr: true,
		},
		{
			name:    "duration exceeds maximum",
			secret:  "0123456789abcdef0123456789abcdef",
			issuer:  "restaurantflow-api",
			ttl:     "25h",
			wantErr: true,
		},
		{
			name:    "whitespace issuer",
			secret:  "0123456789abcdef0123456789abcdef",
			issuer:  "   ",
			ttl:     "15m",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tt.secret)
			t.Setenv("JWT_ISSUER", tt.issuer)
			t.Setenv("JWT_ACCESS_TOKEN_TTL", tt.ttl)

			got, err := loadJWTConfig()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Secret != tt.secret {
				t.Error("JWT secret does not match configured secret")
			}

			if got.Issuer != tt.wantIssuer {
				t.Errorf(
					"issuer = %q, want %q",
					got.Issuer,
					tt.wantIssuer,
				)
			}

			if got.AccessTokenTTL != tt.wantTTL {
				t.Errorf(
					"access token TTL = %v, want %v",
					got.AccessTokenTTL,
					tt.wantTTL,
				)
			}
		})
	}
}
