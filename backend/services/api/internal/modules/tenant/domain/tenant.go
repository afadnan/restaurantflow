package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidTenantID   = errors.New("invalid tenant id")
	ErrInvalidTenantName = errors.New("invalid tenant name")
	ErrInvalidTenantSlug = errors.New("invalid tenant slug")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Tenant struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewTenant(name, slug string) (*Tenant, error) {
	tenant := &Tenant{
		ID:     uuid.New(),
		Name:   strings.TrimSpace(name),
		Slug:   strings.ToLower(strings.TrimSpace(slug)),
		Status: "active",
	}

	if err := tenant.Validate(); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (t *Tenant) Validate() error {
	if t == nil || t.ID == uuid.Nil {
		return ErrInvalidTenantID
	}

	if len(t.Name) < 2 || len(t.Name) > 200 {
		return ErrInvalidTenantName
	}

	if len(t.Slug) < 3 ||
		len(t.Slug) > 100 ||
		!slugPattern.MatchString(t.Slug) {
		return ErrInvalidTenantSlug
	}

	return nil
}
