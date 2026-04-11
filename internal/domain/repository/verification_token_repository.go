package repository

import (
	"context"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type VerificationTokenRepository interface {
	Create(ctx context.Context, token *entity.VerificationToken) (*entity.VerificationToken, error)
	FindByToken(ctx context.Context, token string) (*entity.VerificationToken, error)
	DeleteByEmail(ctx context.Context, email string) error
}
