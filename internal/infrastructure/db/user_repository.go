package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	sqlcgen "github.com/kazumadev619-dev/fishing-api/db/generated"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type userRepository struct {
	queries *sqlcgen.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *userRepository {
	db := stdlib.OpenDBFromPool(pool)
	return &userRepository{queries: sqlcgen.New(db)}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	row, err := r.queries.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toUserEntity(row), nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row, err := r.queries.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toUserEntity(row), nil
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) (*entity.User, error) {
	var passwordHash sql.NullString
	if user.PasswordHash != nil {
		passwordHash = sql.NullString{String: *user.PasswordHash, Valid: true}
	}
	var name sql.NullString
	if user.Name != nil {
		name = sql.NullString{String: *user.Name, Valid: true}
	}

	row, err := r.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: passwordHash,
		Name:         name,
		IsSsoUser:    user.IsSSO,
	})
	if err != nil {
		return nil, err
	}
	return toUserEntity(row), nil
}

func (r *userRepository) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (*entity.User, error) {
	row, err := r.queries.UpdateUserEmailVerified(ctx, sqlcgen.UpdateUserEmailVerifiedParams{
		ID:              id,
		EmailVerifiedAt: sql.NullTime{Time: verifiedAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return toUserEntity(row), nil
}

func toUserEntity(row sqlcgen.User) *entity.User {
	u := &entity.User{
		ID:        row.ID,
		Email:     row.Email,
		IsSSO:     row.IsSsoUser,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.PasswordHash.Valid {
		u.PasswordHash = &row.PasswordHash.String
	}
	if row.Name.Valid {
		u.Name = &row.Name.String
	}
	if row.AvatarUrl.Valid {
		u.AvatarURL = &row.AvatarUrl.String
	}
	if row.EmailVerifiedAt.Valid {
		u.EmailVerifiedAt = &row.EmailVerifiedAt.Time
	}
	return u
}
