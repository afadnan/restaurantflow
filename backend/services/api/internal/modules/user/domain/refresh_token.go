package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const RefreshTokenHashLength = 32

var (
	ErrInvalidRefreshTokenID       = errors.New("invalid refresh token id")
	ErrInvalidRefreshTokenTenantID = errors.New("invalid refresh token tenant id")
	ErrInvalidRefreshTokenUserID   = errors.New("invalid refresh token user id")
	ErrInvalidRefreshTokenFamilyID = errors.New("invalid refresh token family id")
	ErrInvalidRefreshTokenHash     = errors.New("invalid refresh token hash")
	ErrInvalidRefreshTokenExpiry   = errors.New("invalid refresh token expiry")
	ErrInvalidRefreshTokenTime     = errors.New("invalid refresh token creation time")
)

type RefreshToken struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	UserID            uuid.UUID
	TokenHash         []byte
	FamilyID          uuid.UUID
	ExpiresAt         time.Time
	RevokedAt         *time.Time
	ReplacedByTokenID *uuid.UUID
	CreatedAt         time.Time
}

func NewRefreshToken(
	tenantID uuid.UUID,
	userID uuid.UUID,
	familyID uuid.UUID,
	tokenHash []byte,
	expiresAt time.Time,
	now time.Time,
) (*RefreshToken, error) {
	refreshToken := &RefreshToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: append([]byte(nil), tokenHash...),
		FamilyID:  familyID,
		ExpiresAt: expiresAt.UTC(),
		CreatedAt: now.UTC(),
	}

	if err := refreshToken.Validate(); err != nil {
		return nil, err
	}

	return refreshToken, nil
}

func (t *RefreshToken) Validate() error {
	if t == nil || t.ID == uuid.Nil {
		return ErrInvalidRefreshTokenID
	}

	if t.TenantID == uuid.Nil {
		return ErrInvalidRefreshTokenTenantID
	}

	if t.UserID == uuid.Nil {
		return ErrInvalidRefreshTokenUserID
	}

	if t.FamilyID == uuid.Nil {
		return ErrInvalidRefreshTokenFamilyID
	}

	if len(t.TokenHash) != RefreshTokenHashLength {
		return ErrInvalidRefreshTokenHash
	}

	if t.CreatedAt.IsZero() {
		return ErrInvalidRefreshTokenTime
	}

	if t.ExpiresAt.IsZero() || !t.ExpiresAt.After(t.CreatedAt) {
		return ErrInvalidRefreshTokenExpiry
	}

	if t.RevokedAt != nil && t.RevokedAt.Before(t.CreatedAt) {
		return ErrInvalidRefreshTokenTime
	}

	if t.ReplacedByTokenID != nil {
		if *t.ReplacedByTokenID == uuid.Nil ||
			*t.ReplacedByTokenID == t.ID {
			return ErrInvalidRefreshTokenID
		}
	}

	return nil
}

func (t *RefreshToken) IsExpired(now time.Time) bool {
	return t == nil || !now.Before(t.ExpiresAt)
}

func (t *RefreshToken) IsRevoked() bool {
	return t == nil || t.RevokedAt != nil
}

func (t *RefreshToken) IsUsable(now time.Time) bool {
	return t != nil &&
		t.Validate() == nil &&
		!t.IsRevoked() &&
		!t.IsExpired(now)
}
