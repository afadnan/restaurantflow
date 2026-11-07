package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func TestNewTokenService(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		issuer  string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:   "valid configuration",
			secret: testJWTSecret,
			issuer: "restaurantflow-api",
			ttl:    15 * time.Minute,
		},
		{
			name:    "short secret",
			secret:  "short",
			issuer:  "restaurantflow-api",
			ttl:     15 * time.Minute,
			wantErr: true,
		},
		{
			name:    "empty issuer",
			secret:  testJWTSecret,
			issuer:  " ",
			ttl:     15 * time.Minute,
			wantErr: true,
		},
		{
			name:    "zero TTL",
			secret:  testJWTSecret,
			issuer:  "restaurantflow-api",
			ttl:     0,
			wantErr: true,
		},
		{
			name:    "TTL exceeds maximum",
			secret:  testJWTSecret,
			issuer:  "restaurantflow-api",
			ttl:     25 * time.Hour,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewTokenService(
				tt.secret,
				tt.issuer,
				tt.ttl,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if service == nil {
				t.Fatal("expected token service")
			}
		})
	}
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	userID := uuid.New()
	tenantID := uuid.New()

	tokenString, expiresAt, err := service.GenerateAccessToken(
		userID,
		tenantID,
		"owner",
	)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	if tokenString == "" {
		t.Fatal("expected signed token")
	}

	if expiresAt.IsZero() {
		t.Fatal("expected expiration time")
	}

	claims, err := service.ValidateAccessToken(tokenString)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}

	if claims.Subject != userID.String() {
		t.Errorf("subject = %q, want %q", claims.Subject, userID)
	}

	if claims.TenantID != tenantID.String() {
		t.Errorf("tenant ID = %q, want %q", claims.TenantID, tenantID)
	}

	if claims.Role != "owner" {
		t.Errorf("role = %q, want owner", claims.Role)
	}

	if claims.Issuer != "restaurantflow-api" {
		t.Errorf("issuer = %q, want restaurantflow-api", claims.Issuer)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("expected expiration claim")
	}

	if !claims.ExpiresAt.Time.Equal(expiresAt.Truncate(time.Second)) {
		t.Errorf(
			"expiration = %v, want %v",
			claims.ExpiresAt.Time,
			expiresAt.Truncate(time.Second),
		)
	}

	if claims.IssuedAt == nil {
		t.Fatal("expected issued-at claim")
	}
}

func TestValidateAccessTokenRejectsInvalidTokens(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	userID := uuid.New()
	tenantID := uuid.New()

	validToken, _, err := service.GenerateAccessToken(
		userID,
		tenantID,
		"owner",
	)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	tamperedToken := validToken[:len(validToken)-1] + "x"

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not-a-jwt",
		},
		{
			name:  "tampered signature",
			token: tamperedToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateAccessToken(tt.token)

			if err == nil {
				t.Fatal("expected validation error")
			}

			if claims != nil {
				t.Fatal("expected nil claims")
			}
		})
	}
}

func TestValidateAccessTokenRejectsExpiredToken(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	now := time.Now().UTC()

	claims := AccessTokenClaims{
		TenantID: uuid.New().String(),
		Role:     "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "restaurantflow-api",
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	_, err = service.ValidateAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestValidateAccessTokenRejectsWrongIssuer(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	now := time.Now().UTC()

	claims := AccessTokenClaims{
		TenantID: uuid.New().String(),
		Role:     "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "another-service",
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = service.ValidateAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected wrong issuer to be rejected")
	}
}

func TestValidateAccessTokenRejectsUnexpectedAlgorithm(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	now := time.Now().UTC()

	claims := AccessTokenClaims{
		TenantID: uuid.New().String(),
		Role:     "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "restaurantflow-api",
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)

	tokenString, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign unexpected algorithm token: %v", err)
	}

	_, err = service.ValidateAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected unexpected algorithm to be rejected")
	}
}

func TestGenerateAccessTokenRejectsInvalidIdentity(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	tests := []struct {
		name     string
		userID   uuid.UUID
		tenantID uuid.UUID
		role     string
	}{
		{
			name:     "missing user ID",
			userID:   uuid.Nil,
			tenantID: uuid.New(),
			role:     "owner",
		},
		{
			name:     "missing tenant ID",
			userID:   uuid.New(),
			tenantID: uuid.Nil,
			role:     "owner",
		},
		{
			name:     "missing role",
			userID:   uuid.New(),
			tenantID: uuid.New(),
			role:     " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _, err := service.GenerateAccessToken(
				tt.userID,
				tt.tenantID,
				tt.role,
			)

			if err == nil {
				t.Fatal("expected error")
			}

			if token != "" {
				t.Fatal("expected empty token")
			}
		})
	}
}

func TestValidateAccessTokenRejectsMalformedIdentityClaims(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	now := time.Now().UTC()

	tests := []struct {
		name     string
		subject  string
		tenantID string
		role     string
	}{
		{
			name:     "missing subject",
			subject:  "",
			tenantID: uuid.New().String(),
			role:     "owner",
		},
		{
			name:     "invalid subject",
			subject:  "not-a-uuid",
			tenantID: uuid.New().String(),
			role:     "owner",
		},
		{
			name:     "invalid tenant ID",
			subject:  uuid.New().String(),
			tenantID: "not-a-uuid",
			role:     "owner",
		},
		{
			name:     "missing role",
			subject:  uuid.New().String(),
			tenantID: uuid.New().String(),
			role:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := AccessTokenClaims{
				TenantID: tt.tenantID,
				Role:     tt.role,
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "restaurantflow-api",
					Subject:   tt.subject,
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

			tokenString, err := token.SignedString([]byte(testJWTSecret))
			if err != nil {
				t.Fatalf("sign token: %v", err)
			}

			_, err = service.ValidateAccessToken(tokenString)
			if err == nil {
				t.Fatal("expected malformed identity claims to be rejected")
			}
		})
	}
}

func TestValidateAccessTokenRejectsDifferentSecret(t *testing.T) {
	service, err := NewTokenService(
		testJWTSecret,
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}

	userID := uuid.New()
	tenantID := uuid.New()

	tokenString, _, err := service.GenerateAccessToken(
		userID,
		tenantID,
		"owner",
	)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	otherService, err := NewTokenService(
		strings.Repeat("x", 32),
		"restaurantflow-api",
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("create second service: %v", err)
	}

	_, err = otherService.ValidateAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected token signed with another secret to be rejected")
	}
}
