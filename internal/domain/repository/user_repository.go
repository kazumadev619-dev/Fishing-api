package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) (*entity.User, error)
	UpdateEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (*entity.User, error)
}
