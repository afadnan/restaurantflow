package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	userrepository "github.com/afadnan/restaurantflow/services/api/internal/modules/user/repository"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/google/uuid"
)

var ErrInvalidAuthenticatedUser = errors.New("invalid authenticated user")

// AuthenticatedUserValidator resolves the current authorization role
// for an authenticated user.
type AuthenticatedUserValidator interface {
	ValidateAuthenticatedUser(
		ctx context.Context,
		tenantID uuid.UUID,
		userID uuid.UUID,
	) (string, error)
}

type PostgresAuthenticatedUserValidator struct {
	db    *postgres.DB
	users userrepository.UserRepository
}

func NewPostgresAuthenticatedUserValidator(
	db *postgres.DB,
	users userrepository.UserRepository,
) *PostgresAuthenticatedUserValidator {
	return &PostgresAuthenticatedUserValidator{
		db:    db,
		users: users,
	}
}

func (v *PostgresAuthenticatedUserValidator) ValidateAuthenticatedUser(
	ctx context.Context,
	tenantID uuid.UUID,
	userID uuid.UUID,
) (string, error) {
	if v == nil || v.db == nil || v.users == nil {
		return "", errors.New("authenticated user validator is not configured")
	}

	tx, err := v.db.BeginTx(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("begin authenticated user lookup: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	user, err := v.users.GetByID(ctx, tx, tenantID, userID)
	if err != nil {
		if errors.Is(err, userrepository.ErrUserNotFound) {
			return "", ErrInvalidAuthenticatedUser
		}

		return "", fmt.Errorf("load authenticated user: %w", err)
	}

	if user == nil ||
		user.ID != userID ||
		user.TenantID != tenantID ||
		!user.CanLogin() ||
		!user.Role.IsValid() {
		return "", ErrInvalidAuthenticatedUser
	}

	return string(user.Role), nil
}

// AuthenticatedUserMiddleware checks the current database state of the
// authenticated user and replaces the JWT role with the current database role.
// It must run after AuthMiddleware.
func AuthenticatedUserMiddleware(
	validator AuthenticatedUserValidator,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			if validator == nil {
				http.Error(
					w,
					"authentication service unavailable",
					http.StatusServiceUnavailable,
				)
				return
			}

			userID, err := UserIDFromContext(r.Context())
			if err != nil {
				http.Error(
					w,
					ErrAuthenticationRequired.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			tenantID, err := AuthenticatedTenantFromContext(r.Context())
			if err != nil {
				http.Error(
					w,
					ErrAuthenticationRequired.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			role, err := validator.ValidateAuthenticatedUser(
				r.Context(),
				tenantID,
				userID,
			)
			if err != nil {
				if errors.Is(err, ErrInvalidAuthenticatedUser) {
					http.Error(
						w,
						ErrInvalidAuthentication.Error(),
						http.StatusUnauthorized,
					)
					return
				}

				http.Error(
					w,
					"authentication service unavailable",
					http.StatusServiceUnavailable,
				)
				return
			}

			if role == "" {
				http.Error(
					w,
					ErrInvalidAuthentication.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			user, ok := authenticatedUserFromContext(r.Context())
			if !ok {
				http.Error(
					w,
					ErrAuthenticationRequired.Error(),
					http.StatusUnauthorized,
				)
				return
			}

			user.role = role

			ctx := context.WithValue(
				r.Context(),
				authenticatedUserContextKey{},
				user,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
