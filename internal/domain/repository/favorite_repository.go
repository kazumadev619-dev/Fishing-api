package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type FavoriteRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error)
	Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
	Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
	Exists(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) (bool, error)
}
