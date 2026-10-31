package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tenantdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/domain"
	tenantrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/tenant/repository"
	userdomain "github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	userusecase "github.com/afadnan/restaurantflow/services/api/internal/modules/user/usecase"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/auth"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

type failingOwnerRepository struct {
	userrepository.UserRepository
}

var errSimulatedOwnerFailure = errors.New("simulated owner insertion failure")

func (r *failingOwnerRepository) Create(
	ctx context.Context,
	tx *postgres.Tx,
	user *userdomain.User,
) error {
	return errSimulatedOwnerFailure
}

func newOwnerRegistrationUseCase(
	i *integrationDB,
	users userrepository.UserRepository,
) *userusecase.RegisterOwnerUseCase {
	return &userusecase.RegisterOwnerUseCase{
		DB:      i.db,
		Tenants: tenantrepository.NewPostgresTenantRepository(),
		Users:   users,
	}
}

func registrationInput(slug string) userusecase.RegisterOwnerInput {
	return userusecase.RegisterOwnerInput{
		BusinessName: "Integration Cafe",
		BusinessSlug: slug,
		Email:        "owner@example.com",
		Password:     "SecureRegistrationPassword123!",
		FirstName:    "Integration",
	}
}

func countTenantBySlug(
	t *testing.T,
	i *integrationDB,
	slug string,
) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count int
	err := i.pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM tenants WHERE slug = $1",
		slug,
	).Scan(&count)
	require.NoError(t, err)

	return count
}

func TestIntegrationOwnerRegistration(t *testing.T) {
	i := newIntegrationDB(t)
	cleanupPool := newOwnerCleanupPool(t)

	t.Run("successful registration", func(t *testing.T) {
		slug := "owner-success-" + uuid.NewString()
		uc := newOwnerRegistrationUseCase(
			i,
			userrepository.NewPostgresUserRepository(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		result, err := uc.Execute(ctx, registrationInput(slug))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Tenant)
		require.NotNil(t, result.Owner)
		cleanupRegisteredOwner(
			t,
			cleanupPool,
			result.Tenant.ID,
			result.Owner.ID,
		)

		require.Equal(t, result.Tenant.ID, result.Owner.TenantID)
		require.Equal(t, userdomain.RoleOwner, result.Owner.Role)
		require.Equal(t, 1, countTenantBySlug(t, i, slug))

		// Verify the committed owner is visible through its tenant context.
		tx, err := i.db.BeginTx(ctx, result.Tenant.ID)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(context.Background()) }()

		stored, err := userrepository.NewPostgresUserRepository().
			GetByEmail(ctx, tx, result.Tenant.ID, result.Owner.Email)
		require.NoError(t, err)

		require.Equal(t, result.Owner.ID, stored.ID)
		require.Equal(t, userdomain.RoleOwner, stored.Role)

		matches, err := auth.VerifyPassword(
			"SecureRegistrationPassword123!",
			stored.PasswordHash,
		)
		require.NoError(t, err)
		require.True(t, matches)

		require.NotEqual(
			t,
			"SecureRegistrationPassword123!",
			stored.PasswordHash,
		)

		require.NoError(t, tx.Rollback(ctx))
	})

	t.Run("owner insertion failure rolls back tenant", func(t *testing.T) {
		slug := "owner-failure-" + uuid.NewString()

		uc := newOwnerRegistrationUseCase(
			i,
			&failingOwnerRepository{},
		)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := uc.Execute(ctx, registrationInput(slug))
		require.ErrorIs(t, err, errSimulatedOwnerFailure)

		require.Zero(t, countTenantBySlug(t, i, slug))
	})

	t.Run("invalid password creates nothing", func(t *testing.T) {
		slug := "owner-invalid-" + uuid.NewString()
		uc := newOwnerRegistrationUseCase(
			i,
			userrepository.NewPostgresUserRepository(),
		)

		input := registrationInput(slug)
		input.Password = "short"

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := uc.Execute(ctx, input)
		require.ErrorIs(
			t,
			err,
			userusecase.ErrInvalidRegistrationPassword,
		)

		require.Zero(t, countTenantBySlug(t, i, slug))
	})

	t.Run("invalid email creates nothing", func(t *testing.T) {
		slug := "owner-email-" + uuid.NewString()
		uc := newOwnerRegistrationUseCase(
			i,
			userrepository.NewPostgresUserRepository(),
		)

		input := registrationInput(slug)
		input.Email = "invalid-email"

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := uc.Execute(ctx, input)
		require.ErrorIs(t, err, userdomain.ErrInvalidEmail)

		require.Zero(t, countTenantBySlug(t, i, slug))
	})

	t.Run("duplicate business slug rejected", func(t *testing.T) {
		slug := "owner-duplicate-" + uuid.NewString()
		uc := newOwnerRegistrationUseCase(
			i,
			userrepository.NewPostgresUserRepository(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		first, err := uc.Execute(ctx, registrationInput(slug))
		require.NoError(t, err)
		require.NotNil(t, first)
		require.NotNil(t, first.Tenant)
		require.NotNil(t, first.Owner)

		cleanupRegisteredOwner(
			t,
			cleanupPool,
			first.Tenant.ID,
			first.Owner.ID,
		)

		_, err = uc.Execute(ctx, registrationInput(slug))
		require.ErrorIs(t, err, tenantrepository.ErrTenantSlugExists)

		require.Equal(t, 1, countTenantBySlug(t, i, slug))
	})
}

var _ = tenantdomain.ErrInvalidTenantID
