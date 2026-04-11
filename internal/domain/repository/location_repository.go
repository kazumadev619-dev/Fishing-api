package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type LocationRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Location, error)
}
