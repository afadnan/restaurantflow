package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner   Role = "owner"
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleStaff   Role = "staff"
	RoleKitchen Role = "kitchen"
	RoleCashier Role = "cashier"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleManager,
		RoleStaff, RoleKitchen, RoleCashier:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusActive    Status = "active"
	StatusInactive  Status = "inactive"
	StatusSuspended Status = "suspended"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusSuspended:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidUserID   = errors.New("invalid user id")
	ErrInvalidTenantID = errors.New("invalid tenant id")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidName     = errors.New("invalid first name")
	ErrInvalidRole     = errors.New("invalid user role")
	ErrInvalidStatus   = errors.New("invalid user status")
	ErrInvalidHash     = errors.New("invalid password hash")
)

type User struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Email        string
	Phone        *string
	PasswordHash string
	FirstName    string
	LastName     *string
	Role         Role
	Status       Status
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func NewUser(
	tenantID uuid.UUID,
	email string,
	passwordHash string,
	firstName string,
	role Role,
) (*User, error) {
	user := &User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        NormalizeEmail(email),
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(firstName),
		Role:         role,
		Status:       StatusActive,
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Validate() error {
	if u.ID == uuid.Nil {
		return ErrInvalidUserID
	}

	if u.TenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	if len(u.Email) < 3 ||
		len(u.Email) > 254 ||
		strings.ContainsAny(u.Email, " \t\n") ||
		strings.Count(u.Email, "@") != 1 {
		return ErrInvalidEmail
	}

	if len(u.FirstName) == 0 || len(u.FirstName) > 200 {
		return ErrInvalidName
	}

	if u.PasswordHash == "" {
		return ErrInvalidHash
	}

	if !u.Role.IsValid() {
		return ErrInvalidRole
	}

	if !u.Status.IsValid() {
		return ErrInvalidStatus
	}

	return nil
}

func (u *User) CanLogin() bool {
	return u != nil && u.Status == StatusActive
}
