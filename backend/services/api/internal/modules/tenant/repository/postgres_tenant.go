package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type PostgresTenantRepository struct{}

func NewPostgresTenantRepository() *PostgresTenantRepository {
	return &PostgresTenantRepository{}
}

func (r *PostgresTenantRepository) Create(
	ctx context.Context,
	tx *postgres.Tx,
	tenant *domain.Tenant,
) error {
	if tx == nil {
		return ErrInvalidTx
	}

	if err := tenant.Validate(); err != nil {
		return err
	}

	q := db.New(tx.Raw())

	row, err := q.CreateTenant(ctx, db.CreateTenantParams{
		ID: pgtype.UUID{
			Bytes: tenant.ID,
			Valid: true,
		},
		Name: tenant.Name,
		Slug: tenant.Slug,
	})
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "tenants_slug_key" {
			return ErrTenantSlugExists
		}

		return fmt.Errorf("create tenant: %w", err)
	}

	tenant.CreatedAt = row.CreatedAt.Time
	tenant.UpdatedAt = row.UpdatedAt.Time
	tenant.Status = row.Status

	return nil
}

var _ TenantRepository = (*PostgresTenantRepository)(nil)
