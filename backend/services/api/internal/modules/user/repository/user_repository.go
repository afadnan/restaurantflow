package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailExists  = errors.New("email already exists")
	ErrInvalidTx    = errors.New("invalid database transaction")
)

type UserRepository interface {
	Create(
		ctx context.Context,
		tx *postgres.Tx,
		user *domain.User,
	) error

	GetByID(
		ctx context.Context,
		tx *postgres.Tx,
		tenantID uuid.UUID,
		userID uuid.UUID,
	) (*domain.User, error)

	GetByEmail(
		ctx context.Context,
		tx *postgres.Tx,
		tenantID uuid.UUID,
		email string,
	) (*domain.User, error)

	UpdateLastLogin(
		ctx context.Context,
		tx *postgres.Tx,
		tenantID uuid.UUID,
		userID uuid.UUID,
	) (*domain.User, error)
}
