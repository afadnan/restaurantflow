package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

var (
	ErrLogoutNotConfigured = errors.New("logout use case not configured")
	ErrInvalidLogoutInput  = errors.New("invalid logout input")
)

type LogoutUseCase struct {
	DB            *postgres.DB
	RefreshTokens userrepository.RefreshTokenRepository
}

type LogoutInput struct {
	TenantID     uuid.UUID
	RefreshToken string
}

func (uc *LogoutUseCase) Execute(
	ctx context.Context,
	input LogoutInput,
) error {
	if uc == nil ||
		uc.DB == nil ||
		uc.DB.Pool == nil ||
		uc.RefreshTokens == nil {
		return ErrLogoutNotConfigured
	}

	if input.TenantID == uuid.Nil ||
		input.RefreshToken == "" ||
		len(input.RefreshToken) > 512 ||
		strings.TrimSpace(input.RefreshToken) != input.RefreshToken {
		return ErrInvalidLogoutInput
	}

	tokenHash, err := auth.HashRefreshToken(input.RefreshToken)
	if err != nil {
		return ErrInvalidLogoutInput
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return fmt.Errorf("begin logout transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	if err := uc.RefreshTokens.RevokeByHash(
		ctx,
		tx,
		input.TenantID,
		tokenHash,
	); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit logout transaction: %w", err)
	}

	return nil
}
