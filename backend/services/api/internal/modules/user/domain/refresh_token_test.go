package domain

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newValidRefreshToken(t *testing.T) *RefreshToken {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Microsecond)

	token, err := NewRefreshToken(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		bytes.Repeat([]byte{1}, RefreshTokenHashLength),
		now.Add(30*24*time.Hour),
		now,
	)
	if err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	return token
}

func TestNewRefreshToken(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()
	hash := bytes.Repeat([]byte{7}, RefreshTokenHashLength)
	now := time.Now().UTC()

	token, err := NewRefreshToken(
		tenantID,
		userID,
		familyID,
		hash,
		now.Add(30*24*time.Hour),
		now,
	)
	if err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	if token.ID == uuid.Nil {
		t.Fatal("expected generated token ID")
	}

	if token.TenantID != tenantID ||
		token.UserID != userID ||
		token.FamilyID != familyID {
		t.Fatal("refresh token identity mismatch")
	}

	if !bytes.Equal(token.TokenHash, hash) {
		t.Fatal("refresh token hash mismatch")
	}

	if err := token.Validate(); err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
}

func TestNewRefreshTokenCopiesHash(t *testing.T) {
	hash := bytes.Repeat([]byte{5}, RefreshTokenHashLength)
	now := time.Now().UTC()

	token, err := NewRefreshToken(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		hash,
		now.Add(time.Hour),
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	hash[0] = 99

	if token.TokenHash[0] == hash[0] {
		t.Fatal("token hash should not share the caller's backing array")
	}
}

func TestRefreshTokenValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*RefreshToken)
		wantErr error
	}{
		{
			name: "invalid token ID",
			mutate: func(token *RefreshToken) {
				token.ID = uuid.Nil
			},
			wantErr: ErrInvalidRefreshTokenID,
		},
		{
			name: "invalid tenant ID",
			mutate: func(token *RefreshToken) {
				token.TenantID = uuid.Nil
			},
			wantErr: ErrInvalidRefreshTokenTenantID,
		},
		{
			name: "invalid user ID",
			mutate: func(token *RefreshToken) {
				token.UserID = uuid.Nil
			},
			wantErr: ErrInvalidRefreshTokenUserID,
		},
		{
			name: "invalid family ID",
			mutate: func(token *RefreshToken) {
				token.FamilyID = uuid.Nil
			},
			wantErr: ErrInvalidRefreshTokenFamilyID,
		},
		{
			name: "invalid hash length",
			mutate: func(token *RefreshToken) {
				token.TokenHash = []byte("invalid")
			},
			wantErr: ErrInvalidRefreshTokenHash,
		},
		{
			name: "missing creation time",
			mutate: func(token *RefreshToken) {
				token.CreatedAt = time.Time{}
			},
			wantErr: ErrInvalidRefreshTokenTime,
		},
		{
			name: "expiry before creation",
			mutate: func(token *RefreshToken) {
				token.ExpiresAt = token.CreatedAt
			},
			wantErr: ErrInvalidRefreshTokenExpiry,
		},
		{
			name: "revocation before creation",
			mutate: func(token *RefreshToken) {
				revokedAt := token.CreatedAt.Add(-time.Second)
				token.RevokedAt = &revokedAt
			},
			wantErr: ErrInvalidRefreshTokenTime,
		},
		{
			name: "nil replacement ID",
			mutate: func(token *RefreshToken) {
				replacementID := uuid.Nil
				token.ReplacedByTokenID = &replacementID
			},
			wantErr: ErrInvalidRefreshTokenID,
		},
		{
			name: "self replacement",
			mutate: func(token *RefreshToken) {
				replacementID := token.ID
				token.ReplacedByTokenID = &replacementID
			},
			wantErr: ErrInvalidRefreshTokenID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := newValidRefreshToken(t)
			tt.mutate(token)

			err := token.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRefreshTokenExpiry(t *testing.T) {
	token := newValidRefreshToken(t)

	if token.IsExpired(token.CreatedAt) {
		t.Fatal("token should not be expired at creation")
	}

	if !token.IsExpired(token.ExpiresAt) {
		t.Fatal("token should be expired at its expiry time")
	}
}

func TestRefreshTokenRevocation(t *testing.T) {
	token := newValidRefreshToken(t)

	if token.IsRevoked() {
		t.Fatal("new token should not be revoked")
	}

	revokedAt := token.CreatedAt.Add(time.Minute)
	token.RevokedAt = &revokedAt

	if !token.IsRevoked() {
		t.Fatal("token should be revoked")
	}
}

func TestRefreshTokenUsability(t *testing.T) {
	token := newValidRefreshToken(t)

	if !token.IsUsable(token.CreatedAt.Add(time.Minute)) {
		t.Fatal("new token should be usable")
	}

	if token.IsUsable(token.ExpiresAt) {
		t.Fatal("expired token should not be usable")
	}

	revokedAt := token.CreatedAt.Add(time.Minute)
	token.RevokedAt = &revokedAt

	if token.IsUsable(token.CreatedAt.Add(2 * time.Minute)) {
		t.Fatal("revoked token should not be usable")
	}
}

func TestNilRefreshToken(t *testing.T) {
	var token *RefreshToken

	if !errors.Is(token.Validate(), ErrInvalidRefreshTokenID) {
		t.Fatal("nil token should return invalid token ID")
	}

	if !token.IsExpired(time.Now()) {
		t.Fatal("nil token should be treated as expired")
	}

	if !token.IsRevoked() {
		t.Fatal("nil token should be treated as revoked")
	}

	if token.IsUsable(time.Now()) {
		t.Fatal("nil token must not be usable")
	}
}
