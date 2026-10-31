package repository

import (
	"context"
	"errors"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

var (
	ErrTenantSlugExists = errors.New("tenant slug already exists")
	ErrInvalidTx        = errors.New("invalid database transaction")
)

type TenantRepository interface {
	Create(
		ctx context.Context,
		tx *postgres.Tx,
		tenant *domain.Tenant,
	) error
}
