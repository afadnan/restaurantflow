package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/afadnan/restaurantflow/services/api/internal/modules/user/domain"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres"
	"github.com/afadnan/restaurantflow/services/api/internal/platform/postgres/db"
)

type PostgresUserRepository struct{}

func NewPostgresUserRepository() *PostgresUserRepository {
	return &PostgresUserRepository{}
}

func (r *PostgresUserRepository) Create(
	ctx context.Context,
	tx *postgres.Tx,
	user *domain.User,
) error {
	if tx == nil {
		return ErrInvalidTx
	}

	if user == nil {
		return errors.New("user is required")
	}

	if err := user.Validate(); err != nil {
		return err
	}

	q := db.New(tx.Raw())

	_, err := q.CreateUser(ctx, db.CreateUserParams{
		ID:           toPgUUID(user.ID),
		TenantID:     toPgUUID(user.TenantID),
		Email:        user.Email,
		Phone:        toPgText(user.Phone),
		PasswordHash: user.PasswordHash,
		FirstName:    user.FirstName,
		LastName:     toPgText(user.LastName),
		Role:         db.UserRole(user.Role),
		Status:       db.UserStatus(user.Status),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "users_tenant_id_email_key" {
			return ErrEmailExists
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *PostgresUserRepository) GetByID(
	ctx context.Context,
	tx *postgres.Tx,
	tenantID uuid.UUID,
	userID uuid.UUID,
) (*domain.User, error) {
	if tx == nil {
		return nil, ErrInvalidTx
	}

	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if userID == uuid.Nil {
		return nil, domain.ErrInvalidUserID
	}

	q := db.New(tx.Raw())

	row, err := q.GetUserByID(ctx, db.GetUserByIDParams{
		TenantID: toPgUUID(tenantID),
		ID:       toPgUUID(userID),
	})
	if err != nil {
		return nil, handleQueryError("get user by id", err)
	}

	return mapUser(row)
}

func (r *PostgresUserRepository) GetByEmail(
	ctx context.Context,
	tx *postgres.Tx,
	tenantID uuid.UUID,
	email string,
) (*domain.User, error) {
	if tx == nil {
		return nil, ErrInvalidTx
	}

	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	email = domain.NormalizeEmail(email)
	if email == "" {
		return nil, domain.ErrInvalidEmail
	}

	q := db.New(tx.Raw())

	row, err := q.GetUserByEmail(ctx, db.GetUserByEmailParams{
		TenantID: toPgUUID(tenantID),
		Email:    email,
	})
	if err != nil {
		return nil, handleQueryError("get user by email", err)
	}

	return mapUser(row)
}

func (r *PostgresUserRepository) UpdateLastLogin(
	ctx context.Context,
	tx *postgres.Tx,
	tenantID uuid.UUID,
	userID uuid.UUID,
) (*domain.User, error) {
	if tx == nil {
		return nil, ErrInvalidTx
	}

	if tenantID == uuid.Nil {
		return nil, domain.ErrInvalidTenantID
	}

	if userID == uuid.Nil {
		return nil, domain.ErrInvalidUserID
	}

	q := db.New(tx.Raw())

	row, err := q.UpdateUserLastLogin(
		ctx,
		db.UpdateUserLastLoginParams{
			TenantID: toPgUUID(tenantID),
			ID:       toPgUUID(userID),
		},
	)
	if err != nil {
		return nil, handleQueryError("update last login", err)
	}

	return mapUser(row)
}

func handleQueryError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}

	return fmt.Errorf("%s: %w", operation, err)
}

func mapUser(row db.User) (*domain.User, error) {
	user := &domain.User{
		ID:           fromPgUUID(row.ID),
		TenantID:     fromPgUUID(row.TenantID),
		Email:        row.Email,
		Phone:        fromPgText(row.Phone),
		PasswordHash: row.PasswordHash,
		FirstName:    row.FirstName,
		LastName:     fromPgText(row.LastName),
		Role:         domain.Role(row.Role),
		Status:       domain.Status(row.Status),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}

	if row.LastLoginAt.Valid {
		t := row.LastLoginAt.Time
		user.LastLoginAt = &t
	}

	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("map user: %w", err)
	}

	return user, nil
}

func toPgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: value,
		Valid: value != uuid.Nil,
	}
}

func fromPgUUID(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}

	return value.Bytes
}

func toPgText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: *value,
		Valid:  true,
	}
}

func fromPgText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	result := value.String
	return &result
}

var _ UserRepository = (*PostgresUserRepository)(nil)
