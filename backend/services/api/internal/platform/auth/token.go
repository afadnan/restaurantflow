package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidTokenConfig = errors.New("invalid JWT configuration")
	ErrInvalidAccessToken = errors.New("invalid access token")
)

const maxAccessTokenTTL = 24 * time.Hour

// AccessTokenClaims contains the identity and authorization context
// carried by an access token.
type AccessTokenClaims struct {
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`

	jwt.RegisteredClaims
}

// TokenService creates and validates signed access tokens.
type TokenService struct {
	secret         []byte
	issuer         string
	accessTokenTTL time.Duration
	now            func() time.Time
}

// NewTokenService creates a JWT token service.
//
// The secret must contain at least 32 bytes.
// Only HS256 is supported.
func NewTokenService(
	secret string,
	issuer string,
	accessTokenTTL time.Duration,
) (*TokenService, error) {
	secret = strings.TrimSpace(secret)
	issuer = strings.TrimSpace(issuer)

	if len(secret) < 32 ||
		issuer == "" ||
		accessTokenTTL <= 0 ||
		accessTokenTTL > maxAccessTokenTTL {
		return nil, ErrInvalidTokenConfig
	}

	return &TokenService{
		secret:         []byte(secret),
		issuer:         issuer,
		accessTokenTTL: accessTokenTTL,
		now:            time.Now,
	}, nil
}

// GenerateAccessToken creates a signed HS256 access token.
//
// The returned time is the token's expiration time.
func (s *TokenService) GenerateAccessToken(
	userID uuid.UUID,
	tenantID uuid.UUID,
	role string,
) (string, time.Time, error) {
	if s == nil ||
		len(s.secret) < 32 ||
		userID == uuid.Nil ||
		tenantID == uuid.Nil ||
		strings.TrimSpace(role) == "" {
		return "", time.Time{}, ErrInvalidAccessToken
	}

	now := s.now().UTC()
	expiresAt := now.Add(s.accessTokenTTL)

	claims := AccessTokenClaims{
		TenantID: tenantID.String(),
		Role:     strings.TrimSpace(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil
}

// ValidateAccessToken verifies the signature, algorithm, issuer,
// issued-at time, expiration, and required identity claims.
func (s *TokenService) ValidateAccessToken(
	tokenString string,
) (*AccessTokenClaims, error) {
	if s == nil ||
		len(s.secret) < 32 ||
		strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidAccessToken
	}

	claims := &AccessTokenClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			// Explicitly reject every algorithm except HS256.
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidAccessToken
			}

			return s.secret, nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidAccessToken
	}

	if claims.Subject == "" ||
		claims.TenantID == "" ||
		strings.TrimSpace(claims.Role) == "" {
		return nil, ErrInvalidAccessToken
	}

	if _, err := uuid.Parse(claims.Subject); err != nil {
		return nil, ErrInvalidAccessToken
	}

	if _, err := uuid.Parse(claims.TenantID); err != nil {
		return nil, ErrInvalidAccessToken
	}

	return claims, nil
}
