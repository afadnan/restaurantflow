package usecase

import (
	"context"
	"errors"
	"fmt"

	tenantdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	tenantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/repository"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

var (
	ErrRegisterOwnerNotConfigured  = errors.New("owner registration use case not configured")
	ErrInvalidRegistrationPassword = errors.New("invalid registration password")
)

type RegisterOwnerUseCase struct {
	DB      *postgres.DB
	Tenants tenantrepository.TenantRepository
	Users   userrepository.UserRepository
}

type RegisterOwnerInput struct {
	BusinessName string
	BusinessSlug string
	Email        string
	Password     string
	FirstName    string
}

type RegisterOwnerResult struct {
	Tenant *tenantdomain.Tenant
	Owner  *userdomain.User
}

func (uc *RegisterOwnerUseCase) Execute(
	ctx context.Context,
	input RegisterOwnerInput,
) (*RegisterOwnerResult, error) {
	if uc == nil ||
		uc.DB == nil ||
		uc.Tenants == nil ||
		uc.Users == nil {
		return nil, ErrRegisterOwnerNotConfigured
	}

	tenant, err := tenantdomain.NewTenant(
		input.BusinessName,
		input.BusinessSlug,
	)
	if err != nil {
		return nil, err
	}

	// Validate password length before performing expensive hashing.
	// Stronger password policy and breached-password screening can
	// be added before the public endpoint is released.
	if len(input.Password) < 12 || len(input.Password) > 1024 {
		return nil, ErrInvalidRegistrationPassword
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash owner password: %w", err)
	}

	owner, err := userdomain.NewUser(
		tenant.ID,
		input.Email,
		passwordHash,
		input.FirstName,
		userdomain.RoleOwner,
	)
	if err != nil {
		return nil, err
	}

	tx, err := uc.DB.BeginTx(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("begin owner registration transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	if err := uc.Tenants.Create(ctx, tx, tenant); err != nil {
		return nil, fmt.Errorf("register tenant: %w", err)
	}

	if err := uc.Users.Create(ctx, tx, owner); err != nil {
		return nil, fmt.Errorf("register owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit owner registration: %w", err)
	}

	return &RegisterOwnerResult{
		Tenant: tenant,
		Owner:  owner,
	}, nil
}
