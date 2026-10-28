package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := "SecurePassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(hash, password) {
		t.Fatal("hash contains plaintext password")
	}

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("correct password rejected")
	}

	valid, err = VerifyPassword("WrongPassword!", hash)
	if err != nil {
		t.Fatal(err)
	}

	if valid {
		t.Fatal("incorrect password accepted")
	}
}

func TestPasswordUsesUniqueSalt(t *testing.T) {
	first, err := HashPassword("SecurePassword123!")
	if err != nil {
		t.Fatal(err)
	}

	second, err := HashPassword("SecurePassword123!")
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatal("password hashes must use unique salts")
	}
}

func TestInvalidPasswordHash(t *testing.T) {
	valid, err := VerifyPassword(
		"SecurePassword123!",
		"invalid-hash",
	)

	if err == nil || valid {
		t.Fatal("expected invalid password hash error")
	}
}

func TestEmptyPasswordRejected(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Fatal("empty password must be rejected")
	}
}
