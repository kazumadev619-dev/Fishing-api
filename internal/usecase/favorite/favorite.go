package favorite

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
)

// FavoriteUsecase はお気に入り場所の管理ビジネスロジックを担う。
type FavoriteUsecase struct {
	repo repository.FavoriteRepository
}

// NewFavoriteUsecase は FavoriteUsecase を生成する。
func NewFavoriteUsecase(repo repository.FavoriteRepository) *FavoriteUsecase {
	return &FavoriteUsecase{repo: repo}
}

// GetList はユーザーのお気に入り場所一覧を返す。
func (u *FavoriteUsecase) GetList(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	locations, err := u.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing favorites: %w", err)
	}
	return locations, nil
}

// Add はお気に入りを追加する。既に登録済みの場合は ErrAlreadyExists を返す。
func (u *FavoriteUsecase) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	exists, err := u.repo.Exists(ctx, userID, locationID)
	if err != nil {
		return fmt.Errorf("checking favorite existence: %w", err)
	}
	if exists {
		return domain.ErrAlreadyExists
	}
	if err := u.repo.Add(ctx, userID, locationID); err != nil {
		return fmt.Errorf("adding favorite: %w", err)
	}
	return nil
}

// Delete はお気に入りを削除する。存在しない場合は ErrNotFound を返す。
func (u *FavoriteUsecase) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	if err := u.repo.Delete(ctx, userID, locationID); err != nil {
		return fmt.Errorf("deleting favorite: %w", err)
	}
	return nil
}
