package db

import (
	"context"
	"database/sql"
	"errors"

	sqlcgen "github.com/kazumadev619-dev/fishing-api/db/generated"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
)

type verificationTokenRepository struct {
	queries *sqlcgen.Queries
}

func NewVerificationTokenRepository(db *sql.DB) repository.VerificationTokenRepository {
	return &verificationTokenRepository{queries: sqlcgen.New(db)}
}

func (r *verificationTokenRepository) Create(ctx context.Context, token *entity.VerificationToken) (*entity.VerificationToken, error) {
	row, err := r.queries.CreateVerificationToken(ctx, sqlcgen.CreateVerificationTokenParams{
		ID:        token.ID,
		Email:     token.Email,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return &entity.VerificationToken{
		ID:        row.ID,
		Email:     row.Email,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *verificationTokenRepository) FindByToken(ctx context.Context, token string) (*entity.VerificationToken, error) {
	row, err := r.queries.FindVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &entity.VerificationToken{
		ID:        row.ID,
		Email:     row.Email,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *verificationTokenRepository) DeleteByEmail(ctx context.Context, email string) error {
	return r.queries.DeleteVerificationTokensByEmail(ctx, email)
}
