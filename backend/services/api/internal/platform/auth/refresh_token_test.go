package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
)

func TestGenerateRefreshToken(t *testing.T) {
	generator := NewRefreshTokenGenerator()

	generated, err := generator.Generate()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	if generated == nil {
		t.Fatal("expected generated refresh token")
	}

	if generated.Plaintext == "" {
		t.Fatal("expected plaintext token")
	}

	if len(generated.Hash) != sha256.Size {
		t.Fatalf("hash length = %d, want %d", len(generated.Hash), sha256.Size)
	}

	randomBytes, err := base64.RawURLEncoding.DecodeString(generated.Plaintext)
	if err != nil {
		t.Fatalf("decode plaintext token: %v", err)
	}

	if len(randomBytes) != RefreshTokenRandomBytes {
		t.Fatalf(
			"random bytes length = %d, want %d",
			len(randomBytes),
			RefreshTokenRandomBytes,
		)
	}

	expectedHash := sha256.Sum256([]byte(generated.Plaintext))

	if !bytes.Equal(generated.Hash, expectedHash[:]) {
		t.Fatal("stored hash does not match plaintext token hash")
	}
}

func TestGenerateRefreshTokenProducesUniqueTokens(t *testing.T) {
	generator := NewRefreshTokenGenerator()

	first, err := generator.Generate()
	if err != nil {
		t.Fatalf("generate first token: %v", err)
	}

	second, err := generator.Generate()
	if err != nil {
		t.Fatalf("generate second token: %v", err)
	}

	if first.Plaintext == second.Plaintext {
		t.Fatal("expected unique plaintext tokens")
	}

	if bytes.Equal(first.Hash, second.Hash) {
		t.Fatal("expected unique token hashes")
	}
}

func TestHashRefreshToken(t *testing.T) {
	plaintext := "example-refresh-token"

	hash, err := HashRefreshToken(plaintext)
	if err != nil {
		t.Fatalf("hash refresh token: %v", err)
	}

	expected := sha256.Sum256([]byte(plaintext))

	if !bytes.Equal(hash, expected[:]) {
		t.Fatal("unexpected SHA-256 hash")
	}
}

func TestHashRefreshTokenRejectsEmptyToken(t *testing.T) {
	hash, err := HashRefreshToken("")

	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidRefreshToken)
	}

	if hash != nil {
		t.Fatal("expected nil hash")
	}
}

func TestNilRefreshTokenGenerator(t *testing.T) {
	var generator *RefreshTokenGenerator

	generated, err := generator.Generate()

	if !errors.Is(err, ErrRefreshTokenGeneration) {
		t.Fatalf("error = %v, want %v", err, ErrRefreshTokenGeneration)
	}

	if generated != nil {
		t.Fatal("expected nil generated token")
	}
}
