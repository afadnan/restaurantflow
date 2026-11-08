package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

type RefreshTokenRepository interface {
	Create(
		ctx context.Context,
		tx *postgres.Tx,
		token *domain.RefreshToken,
	) error

	GetByHash(
		ctx context.Context,
		tx *postgres.Tx,
		tenantID uuid.UUID,
		tokenHash []byte,
	) (*domain.RefreshToken, error)

	LockByHash(
		ctx context.Context,
		tx *postgres.Tx,
		tenantID uuid.UUID,
		tokenHash []byte,
	) (*domain.RefreshToken, error)

	Rotate(
		ctx context.Context,
		tx *postgres.Tx,
		tenantID uuid.UUID,
		tokenID uuid.UUID,
		replacementTokenID uuid.UUID,
	) error

	RevokeFamily(
		ctx context.Context,
		tx *postgres.Tx,
		tenantID uuid.UUID,
		familyID uuid.UUID,
	) error
}
