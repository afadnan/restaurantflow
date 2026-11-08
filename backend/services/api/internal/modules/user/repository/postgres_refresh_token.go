package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type PostgresRefreshTokenRepository struct{}

func NewPostgresRefreshTokenRepository() *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{}
}

func (r *PostgresRefreshTokenRepository) Create(
	ctx context.Context,
	tx *postgres.Tx,
	token *domain.RefreshToken,
) error {
	if tx == nil {
		return ErrInvalidTx
	}

	if token == nil {
		return errors.New("refresh token is required")
	}

	if err := token.Validate(); err != nil {
		return err
	}

	q := db.New(tx.Raw())

	_, err := q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:        toPgUUID(token.ID),
		TenantID:  toPgUUID(token.TenantID),
		UserID:    toPgUUID(token.UserID),
		TokenHash: token.TokenHash,
		FamilyID:  toPgUUID(token.FamilyID),
		ExpiresAt: toPgTimestamptz(token.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func toPgTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}

func (r *PostgresRefreshTokenRepository) GetByHash(
	ctx context.Context,
	tx *postgres.Tx,
	tenantID uuid.UUID,
	tokenHash []byte,
) (*domain.RefreshToken, error) {
	if tx == nil {
		return nil, ErrInvalidTx
	}

	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidRefreshTokenTenantID
	}

	if len(tokenHash) != domain.RefreshTokenHashLength {
		return nil, domain.ErrInvalidRefreshTokenHash
	}

	q := db.New(tx.Raw())

	row, err := q.GetRefreshTokenByHash(ctx, db.GetRefreshTokenByHashParams{
		TenantID:  toPgUUID(tenantID),
		TokenHash: tokenHash,
	})
	if err != nil {
		return nil, handleRefreshTokenQueryError("get refresh token by hash", err)
	}

	return mapRefreshToken(row)
}

func (r *PostgresRefreshTokenRepository) LockByHash(
	ctx context.Context,
	tx *postgres.Tx,
	tenantID uuid.UUID,
	tokenHash []byte,
) (*domain.RefreshToken, error) {
	if tx == nil {
		return nil, ErrInvalidTx
	}

	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidRefreshTokenTenantID
	}

	if len(tokenHash) != domain.RefreshTokenHashLength {
		return nil, domain.ErrInvalidRefreshTokenHash
	}

	q := db.New(tx.Raw())

	row, err := q.LockRefreshTokenByHash(ctx, db.LockRefreshTokenByHashParams{
		TenantID:  toPgUUID(tenantID),
		TokenHash: tokenHash,
	})
	if err != nil {
		return nil, handleRefreshTokenQueryError("lock refresh token by hash", err)
	}

	return mapRefreshToken(row)
}

func (r *PostgresRefreshTokenRepository) Rotate(
	ctx context.Context,
	tx *postgres.Tx,
	tenantID uuid.UUID,
	tokenID uuid.UUID,
	replacementTokenID uuid.UUID,
) error {
	if tx == nil {
		return ErrInvalidTx
	}

	if tenantID == uuid.Nil {
		return domain.ErrInvalidRefreshTokenTenantID
	}

	if tokenID == uuid.Nil || replacementTokenID == uuid.Nil ||
		tokenID == replacementTokenID {
		return domain.ErrInvalidRefreshTokenID
	}

	q := db.New(tx.Raw())

	_, err := q.RotateRefreshToken(ctx, db.RotateRefreshTokenParams{
		TenantID:          toPgUUID(tenantID),
		ID:                toPgUUID(tokenID),
		ReplacedByTokenID: toPgUUID(replacementTokenID),
	})
	if err != nil {
		return handleRefreshTokenQueryError("rotate refresh token", err)
	}

	return nil
}

func (r *PostgresRefreshTokenRepository) RevokeFamily(
	ctx context.Context,
	tx *postgres.Tx,
	tenantID uuid.UUID,
	familyID uuid.UUID,
) error {
	if tx == nil {
		return ErrInvalidTx
	}

	if tenantID == uuid.Nil {
		return domain.ErrInvalidRefreshTokenTenantID
	}

	if familyID == uuid.Nil {
		return domain.ErrInvalidRefreshTokenFamilyID
	}

	q := db.New(tx.Raw())

	if err := q.RevokeRefreshTokenFamily(ctx, db.RevokeRefreshTokenFamilyParams{
		TenantID: toPgUUID(tenantID),
		FamilyID: toPgUUID(familyID),
	}); err != nil {
		return fmt.Errorf("revoke refresh token family: %w", err)
	}

	return nil
}

func handleRefreshTokenQueryError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRefreshTokenNotFound
	}

	return fmt.Errorf("%s: %w", operation, err)
}

func mapRefreshToken(row db.RefreshToken) (*domain.RefreshToken, error) {
	token := &domain.RefreshToken{
		ID:        fromPgUUID(row.ID),
		TenantID:  fromPgUUID(row.TenantID),
		UserID:    fromPgUUID(row.UserID),
		TokenHash: append([]byte(nil), row.TokenHash...),
		FamilyID:  fromPgUUID(row.FamilyID),
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}

	if row.RevokedAt.Valid {
		revokedAt := row.RevokedAt.Time
		token.RevokedAt = &revokedAt
	}

	if row.ReplacedByTokenID.Valid {
		replacementID := fromPgUUID(row.ReplacedByTokenID)
		token.ReplacedByTokenID = &replacementID
	}

	if err := token.Validate(); err != nil {
		return nil, fmt.Errorf("map refresh token: %w", err)
	}

	return token, nil
}

var _ RefreshTokenRepository = (*PostgresRefreshTokenRepository)(nil)
