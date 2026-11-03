package repository

import (
	"context"
	"errors"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

var (
	ErrTenantSlugExists = errors.New("tenant slug already exists")
	ErrTenantNotFound   = errors.New("tenant not found")
	ErrInvalidTx        = errors.New("invalid database transaction")
	ErrInvalidDatabase  = errors.New("invalid database connection")
)

type TenantRepository interface {
	Create(
		ctx context.Context,
		tx *postgres.Tx,
		tenant *domain.Tenant,
	) error

	GetBySlug(
		ctx context.Context,
		database *postgres.DB,
		slug string,
	) (*domain.Tenant, error)
}
