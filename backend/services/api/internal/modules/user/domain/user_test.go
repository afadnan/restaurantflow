package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	tenantID := uuid.New()

	user, err := NewUser(
		tenantID,
		" TEST@Example.COM ",
		"hashed-password",
		" Test ",
		RoleStaff,
	)
	if err != nil {
		t.Fatal(err)
	}

	if user.TenantID != tenantID {
		t.Fatal("tenant ID mismatch")
	}

	if user.Email != "test@example.com" {
		t.Fatalf("unexpected email: %s", user.Email)
	}

	if user.FirstName != "Test" {
		t.Fatal("first name not normalized")
	}

	if !user.CanLogin() {
		t.Fatal("active user should be able to log in")
	}
}

func TestInvalidTenant(t *testing.T) {
	_, err := NewUser(
		uuid.Nil,
		"test@example.com",
		"hashed-password",
		"Test",
		RoleStaff,
	)

	if !errors.Is(err, ErrInvalidTenantID) {
		t.Fatalf("expected invalid tenant ID, got %v", err)
	}
}

func TestInvalidRole(t *testing.T) {
	_, err := NewUser(
		uuid.New(),
		"test@example.com",
		"hashed-password",
		"Test",
		Role("superadmin"),
	)

	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected invalid role, got %v", err)
	}
}

func TestSuspendedUserCannotLogin(t *testing.T) {
	user, err := NewUser(
		uuid.New(),
		"test@example.com",
		"hashed-password",
		"Test",
		RoleStaff,
	)
	if err != nil {
		t.Fatal(err)
	}

	user.Status = StatusSuspended

	if user.CanLogin() {
		t.Fatal("suspended user must not log in")
	}
}
