package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	if tenant == nil {
		return errors.New("tenant is required")
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

func (r *PostgresTenantRepository) GetBySlug(
	ctx context.Context,
	database *postgres.DB,
	slug string,
) (*domain.Tenant, error) {
	if database == nil || database.Pool == nil {
		return nil, ErrInvalidDatabase
	}

	slug = strings.ToLower(strings.TrimSpace(slug))

	if len(slug) < 3 || len(slug) > 100 {
		return nil, domain.ErrInvalidTenantSlug
	}

	q := db.New(database.Pool)

	row, err := q.GetTenantBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}

		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}

	tenant := &domain.Tenant{
		ID:        uuid.UUID(row.ID.Bytes),
		Name:      row.Name,
		Slug:      row.Slug,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if err := tenant.Validate(); err != nil {
		return nil, fmt.Errorf("map tenant: %w", err)
	}

	return tenant, nil
}

var _ TenantRepository = (*PostgresTenantRepository)(nil)
