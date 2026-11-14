package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

var (
	ErrRefreshNotConfigured = errors.New("refresh use case not configured")
	ErrInvalidRefreshInput  = errors.New("invalid refresh input")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
)

type RefreshUseCase struct {
	DB                    *postgres.DB
	Users                 userrepository.UserRepository
	RefreshTokens         userrepository.RefreshTokenRepository
	Tokens                AccessTokenIssuer
	RefreshTokenGenerator RefreshTokenGenerator
}

type RefreshInput struct {
	TenantID     uuid.UUID
	RefreshToken string
}

type RefreshResult struct {
	AccessToken           string
	TokenType             string
	ExpiresAt             time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	User                  *userdomain.User
}

func (uc *RefreshUseCase) Execute(
	ctx context.Context,
	input RefreshInput,
) (*RefreshResult, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.DB.Pool == nil ||
		uc.Users == nil ||
		uc.RefreshTokens == nil ||
		uc.Tokens == nil ||
		uc.RefreshTokenGenerator == nil {
		return nil, ErrRefreshNotConfigured
	}

	// The tenant ID establishes the RLS context. It is not proof
	// of identity; the presented token must match a stored hash.
	if input.TenantID == uuid.Nil ||
		input.RefreshToken == "" ||
		len(input.RefreshToken) > 512 ||
		strings.TrimSpace(input.RefreshToken) != input.RefreshToken {
		return nil, ErrInvalidRefreshInput
	}

	tokenHash, err := auth.HashRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	tx, err := uc.DB.BeginTx(ctx, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("begin refresh transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Lock the presented token so concurrent refresh attempts cannot
	// successfully rotate the same token.
	currentToken, err := uc.RefreshTokens.LockByHash(
		ctx,
		tx,
		input.TenantID,
		tokenHash,
	)
	if err != nil {
		if errors.Is(err, userrepository.ErrRefreshTokenNotFound) {
			return nil, ErrInvalidRefreshToken
		}

		return nil, fmt.Errorf("lock refresh token: %w", err)
	}

	if currentToken == nil ||
		currentToken.TenantID != input.TenantID {
		return nil, ErrInvalidRefreshToken
	}

	// A revoked token with a replacement indicates that a previously
	// rotated token is being reused. Revoke the whole family.
	//
	// This transaction MUST commit before returning the unauthorized
	// result, otherwise the deferred rollback would undo the revocation.
	if currentToken.IsRevoked() {
		if currentToken.ReplacedByTokenID != nil {
			if err := uc.RefreshTokens.RevokeFamily(
				ctx,
				tx,
				input.TenantID,
				currentToken.FamilyID,
			); err != nil {
				return nil, fmt.Errorf(
					"revoke reused refresh token family: %w",
					err,
				)
			}

			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf(
					"commit refresh token family revocation: %w",
					err,
				)
			}
		}

		return nil, ErrInvalidRefreshToken
	}

	now := time.Now().UTC()

	if currentToken.IsExpired(now) {
		return nil, ErrInvalidRefreshToken
	}

	user, err := uc.Users.GetByID(
		ctx,
		tx,
		input.TenantID,
		currentToken.UserID,
	)
	if err != nil {
		if errors.Is(err, userrepository.ErrUserNotFound) {
			return nil, ErrInvalidRefreshToken
		}

		return nil, fmt.Errorf("load refresh token user: %w", err)
	}

	if user == nil ||
		user.ID != currentToken.UserID ||
		user.TenantID != input.TenantID ||
		!user.CanLogin() {
		return nil, ErrInvalidRefreshToken
	}

	accessToken, accessTokenExpiresAt, err := uc.Tokens.GenerateAccessToken(
		user.ID,
		input.TenantID,
		string(user.Role),
	)
	if err != nil {
		return nil, fmt.Errorf("generate refreshed access token: %w", err)
	}

	generatedRefreshToken, err := uc.RefreshTokenGenerator.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate replacement refresh token: %w", err)
	}

	if generatedRefreshToken == nil ||
		generatedRefreshToken.Plaintext == "" ||
		len(generatedRefreshToken.Hash) != auth.RefreshTokenRandomBytes {
		return nil, errors.New("refresh token generator returned invalid token")
	}

	refreshTokenExpiresAt := now.Add(RefreshTokenTTL)

	replacementToken, err := userdomain.NewRefreshToken(
		input.TenantID,
		user.ID,
		currentToken.FamilyID,
		generatedRefreshToken.Hash,
		refreshTokenExpiresAt,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create replacement refresh token: %w", err)
	}

	// Insert the replacement and revoke the old token atomically.
	if err := uc.RefreshTokens.Create(ctx, tx, replacementToken); err != nil {
		return nil, fmt.Errorf("persist replacement refresh token: %w", err)
	}

	if err := uc.RefreshTokens.Rotate(
		ctx,
		tx,
		input.TenantID,
		currentToken.ID,
		replacementToken.ID,
	); err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh transaction: %w", err)
	}

	return &RefreshResult{
		AccessToken:           accessToken,
		TokenType:             "Bearer",
		ExpiresAt:             accessTokenExpiresAt,
		RefreshToken:          generatedRefreshToken.Plaintext,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
		User:                  user,
	}, nil
}
