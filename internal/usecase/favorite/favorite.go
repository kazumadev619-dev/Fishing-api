package favorite

import (
	"context"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
)

type FavoriteUsecase struct {
	repo repository.FavoriteRepository
}

func NewFavoriteUsecase(repo repository.FavoriteRepository) *FavoriteUsecase {
	return &FavoriteUsecase{repo: repo}
}

func (u *FavoriteUsecase) GetList(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	return u.repo.FindByUserID(ctx, userID)
}

func (u *FavoriteUsecase) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	exists, err := u.repo.Exists(ctx, userID, locationID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrAlreadyExists
	}
	return u.repo.Add(ctx, userID, locationID)
}

func (u *FavoriteUsecase) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	return u.repo.Delete(ctx, userID, locationID)
}
