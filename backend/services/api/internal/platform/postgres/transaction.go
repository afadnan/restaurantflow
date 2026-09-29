package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
)

var ErrTenantIDRequired = errors.New("tenant id is required")

type DB struct {
	Pool *pgxpool.Pool
}

type Tx struct {
	tx pgx.Tx
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{
		Pool: pool,
	}
}

// BeginTx starts a PostgreSQL transaction and establishes the tenant
// context for the lifetime of that transaction.
//
// The tenant context is stored using PostgreSQL's transaction-local
// configuration mechanism:
//
//	SET LOCAL app.tenant_id = '<tenant-id>'
//
// This allows PostgreSQL Row-Level Security policies to enforce tenant
// isolation for every query executed through this transaction.
func (db *DB) BeginTx(
	ctx context.Context,
	tenantID uuid.UUID,
) (*Tx, error) {
	if tenantID == uuid.Nil {
		return nil, ErrTenantIDRequired
	}

	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin postgres transaction: %w", err)
	}

	if err := setTenantID(ctx, tx, tenantID); err != nil {
		_ = tx.Rollback(ctx)

		return nil, fmt.Errorf(
			"initialize postgres tenant context: %w",
			err,
		)
	}

	return &Tx{
		tx: tx,
	}, nil
}

// BeginWorkerTx starts a PostgreSQL transaction without establishing
// a tenant context.
//
// Platform workers such as the transactional outbox dispatcher operate
// across all tenants. They therefore must not use BeginTx, which sets
// app.tenant_id and activates tenant-scoped RLS visibility.
//
// The database role used by this worker connection must have the
// appropriate privileges to read and update outbox_events across tenants.
func (db *DB) BeginWorkerTx(
	ctx context.Context,
) (*Tx, error) {
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"begin postgres worker transaction: %w",
			err,
		)
	}

	return &Tx{
		tx: tx,
	}, nil
}

// setTenantID establishes the tenant identifier as a transaction-local
// PostgreSQL setting.
//
// The third argument to set_config is true, which is equivalent to
// SET LOCAL. Therefore the tenant context disappears automatically when
// the transaction commits or rolls back.
func setTenantID(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
) error {
	if tenantID == uuid.Nil {
		return ErrTenantIDRequired
	}

	_, err := tx.Exec(
		ctx,
		"SELECT set_config('app.tenant_id', $1, true)",
		tenantID.String(),
	)
	if err != nil {
		return fmt.Errorf("set postgres tenant context: %w", err)
	}

	return nil
}

func (tx *Tx) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}

func (tx *Tx) Rollback(ctx context.Context) error {
	return tx.tx.Rollback(ctx)
}

func (tx *Tx) Raw() pgx.Tx {
	return tx.tx
}

var _ domain.Transaction = (*Tx)(nil)
