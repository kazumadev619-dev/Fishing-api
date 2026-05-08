package location

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

// MapsAPI は外部地図APIクライアントのインターフェース。
// infrastructure/external/ の実装が満たすこと。
type MapsAPI interface {
	SearchLocations(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error)
}

// Cache はキャッシュストアのインターフェース。
// infrastructure/cache/ の実装が満たすこと。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

const locationCacheTTL = 24 * time.Hour

// LocationUsecase は場所検索のビジネスロジックを担う。
// Redisキャッシュを使い、キャッシュミス時のみ外部APIを呼び出す。
type LocationUsecase struct {
	mapsAPI MapsAPI
	cache   Cache
}

// NewLocationUsecase は LocationUsecase を生成する。
func NewLocationUsecase(mapsAPI MapsAPI, cache Cache) *LocationUsecase {
	return &LocationUsecase{mapsAPI: mapsAPI, cache: cache}
}

// Search は指定クエリで場所を検索し、結果を返す。
// キャッシュにデータがある場合はAPIを呼ばずキャッシュから返す。
func (u *LocationUsecase) Search(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error) {
	h := sha256.Sum256([]byte(query))
	key := fmt.Sprintf("location:%x:%d", h, limit)

	cached, _ := u.cache.Get(ctx, key) //nolint:errcheck // cache miss is acceptable
	if cached != nil {
		var results []*entity.LocationSearchResult
		if err := json.Unmarshal(cached, &results); err == nil {
			return results, nil
		}
	}

	results, err := u.mapsAPI.SearchLocations(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("searching locations for query %q: %w", query, err)
	}

	if b, err := json.Marshal(results); err == nil {
		u.cache.Set(ctx, key, b, locationCacheTTL) //nolint:errcheck
	}

	return results, nil
}
