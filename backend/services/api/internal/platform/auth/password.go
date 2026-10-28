package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	saltLength   = 16
)

var ErrInvalidPasswordHash = errors.New("invalid password hash")

func HashPassword(password string) (string, error) {
	if len(password) == 0 || len(password) > 1024 {
		return "", errors.New("invalid password length")
	}

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	encoder := base64.RawStdEncoding

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		encoder.EncodeToString(salt),
		encoder.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 ||
		parts[0] != "" ||
		parts[1] != "argon2id" ||
		parts[2] != "v=19" {
		return false, ErrInvalidPasswordHash
	}

	var memory, timeCost, threads uint32

	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&memory,
		&timeCost,
		&threads,
	); err != nil {
		return false, ErrInvalidPasswordHash
	}

	// Bound parameters before performing expensive work.
	if memory != argonMemory ||
		timeCost != argonTime ||
		threads != argonThreads {
		return false, ErrInvalidPasswordHash
	}

	encoder := base64.RawStdEncoding

	salt, err := encoder.DecodeString(parts[4])
	if err != nil || len(salt) != saltLength {
		return false, ErrInvalidPasswordHash
	}

	expected, err := encoder.DecodeString(parts[5])
	if err != nil || len(expected) != argonKeyLen {
		return false, ErrInvalidPasswordHash
	}

	actual := argon2.IDKey(
		[]byte(password),
		salt,
		timeCost,
		memory,
		uint8(threads),
		uint32(len(expected)),
	)

	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
