package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/order/domain"
)

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

func (db *DB) BeginTx(
	ctx context.Context,
) (*Tx, error) {
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	return &Tx{tx: tx}, nil
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
