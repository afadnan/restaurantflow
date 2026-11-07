package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	tenantdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	tenantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/repository"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

var (
	ErrLoginNotConfigured = errors.New("login use case not configured")
	ErrInvalidLoginInput  = errors.New("invalid login input")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type AccessTokenIssuer interface {
	GenerateAccessToken(
		userID uuid.UUID,
		tenantID uuid.UUID,
		role string,
	) (string, time.Time, error)
}

type LoginUseCase struct {
	DB      *postgres.DB
	Tenants tenantrepository.TenantRepository
	Users   userrepository.UserRepository
	Tokens  AccessTokenIssuer
}

type LoginInput struct {
	BusinessSlug string
	Email        string
	Password     string
}

type LoginResult struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
	Tenant      *tenantdomain.Tenant
	User        *userdomain.User
}

func (uc *LoginUseCase) Execute(
	ctx context.Context,
	input LoginInput,
) (*LoginResult, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.DB.Pool == nil ||
		uc.Tenants == nil ||
		uc.Users == nil ||
		uc.Tokens == nil {
		return nil, ErrLoginNotConfigured
	}

	slug := strings.ToLower(strings.TrimSpace(input.BusinessSlug))
	email := userdomain.NormalizeEmail(input.Email)

	if len(slug) < 3 ||
		len(slug) > 100 ||
		len(email) < 3 ||
		len(email) > 254 ||
		strings.Count(email, "@") != 1 ||
		strings.ContainsAny(email, " \t\n") ||
		len(input.Password) == 0 ||
		len(input.Password) > 1024 {
		return nil, ErrInvalidLoginInput
	}

	// Tenant lookup happens before the transaction because we need
	// the tenant ID to establish PostgreSQL's RLS context.
	tenant, err := uc.Tenants.GetBySlug(ctx, uc.DB, slug)
	if err != nil {
		if errors.Is(err, tenantrepository.ErrTenantNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("lookup login tenant: %w", err)
	}

	if tenant == nil || tenant.Status != "active" {
		return nil, ErrInvalidCredentials
	}

	tx, err := uc.DB.BeginTx(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("begin login transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	user, err := uc.Users.GetByEmail(ctx, tx, tenant.ID, email)
	if err != nil {
		if errors.Is(err, userrepository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("lookup login user: %w", err)
	}

	if user == nil ||
		user.TenantID != tenant.ID ||
		!user.CanLogin() {
		return nil, ErrInvalidCredentials
	}

	passwordMatches, err := auth.VerifyPassword(
		input.Password,
		user.PasswordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("verify login password: %w", err)
	}

	if !passwordMatches {
		return nil, ErrInvalidCredentials
	}

	updatedUser, err := uc.Users.UpdateLastLogin(
		ctx,
		tx,
		tenant.ID,
		user.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update user last login: %w", err)
	}

	if updatedUser == nil ||
		updatedUser.ID != user.ID ||
		updatedUser.TenantID != tenant.ID ||
		!updatedUser.CanLogin() {
		return nil, errors.New("invalid user returned after last login update")
	}

	accessToken, expiresAt, err := uc.Tokens.GenerateAccessToken(
		updatedUser.ID,
		tenant.ID,
		string(updatedUser.Role),
	)
	if err != nil {
		return nil, fmt.Errorf("generate login access token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit login transaction: %w", err)
	}

	return &LoginResult{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		Tenant:      tenant,
		User:        updatedUser,
	}, nil
}
