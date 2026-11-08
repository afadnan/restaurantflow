package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

const (
	// RefreshTokenRandomBytes provides 256 bits of cryptographic entropy.
	RefreshTokenRandomBytes = 32
)

var (
	ErrRefreshTokenGeneration = errors.New("failed to generate refresh token")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
)

// GeneratedRefreshToken contains the plaintext token for the client
// and its SHA-256 hash for database persistence.
//
// Never persist or log Plaintext.
type GeneratedRefreshToken struct {
	Plaintext string
	Hash      []byte
}

// RefreshTokenGenerator generates opaque refresh tokens.
type RefreshTokenGenerator struct{}

// NewRefreshTokenGenerator creates a refresh-token generator.
func NewRefreshTokenGenerator() *RefreshTokenGenerator {
	return &RefreshTokenGenerator{}
}

// Generate creates a cryptographically secure opaque refresh token.
//
// The plaintext is URL-safe and contains 256 bits of random entropy.
// Only its SHA-256 hash should be stored in the database.
func (g *RefreshTokenGenerator) Generate() (*GeneratedRefreshToken, error) {
	if g == nil {
		return nil, ErrRefreshTokenGeneration
	}

	randomBytes := make([]byte, RefreshTokenRandomBytes)

	if _, err := rand.Read(randomBytes); err != nil {
		return nil, ErrRefreshTokenGeneration
	}

	plaintext := base64.RawURLEncoding.EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(plaintext))

	return &GeneratedRefreshToken{
		Plaintext: plaintext,
		Hash:      hash[:],
	}, nil
}

// HashRefreshToken hashes a plaintext refresh token using SHA-256.
func HashRefreshToken(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, ErrInvalidRefreshToken
	}

	hash := sha256.Sum256([]byte(plaintext))
	return hash[:], nil
}
