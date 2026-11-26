package tide

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

// TideAPI は外部潮汐APIクライアントのインターフェース。
// infrastructure/external/ の実装が満たすこと。
type TideAPI interface {
	FetchTideData(ctx context.Context, prefCode, portCode, date, rng string) (*entity.TideData, error)
}

// Cache はキャッシュストアのインターフェース。
// infrastructure/cache/ の実装が満たすこと。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

const tideTTL = 6 * time.Hour

// TideUsecase は潮汐情報の取得ビジネスロジックを担う。
// Redisキャッシュを使い、キャッシュミス時のみ外部APIを呼び出す。
type TideUsecase struct {
	api   TideAPI
	cache Cache
}

// NewTideUsecase は TideUsecase を生成する。
func NewTideUsecase(api TideAPI, cache Cache) *TideUsecase {
	return &TideUsecase{api: api, cache: cache}
}

// GetTideData は港・日付・範囲（day / week / month）の潮汐と、キャッシュから返したかを返す。
// 取得に失敗したときはキャッシュに書かない。
func (u *TideUsecase) GetTideData(ctx context.Context, prefCode, portCode, date, rng string) (*entity.TideData, bool, error) {
	key := fmt.Sprintf("tide:%s:%s:%s:%s", prefCode, portCode, date, rng)

	cached, _ := u.cache.Get(ctx, key) //nolint:errcheck // cache miss is acceptable
	if cached != nil {
		var data entity.TideData
		if err := json.Unmarshal(cached, &data); err == nil {
			return &data, true, nil
		}
	}

	data, err := u.api.FetchTideData(ctx, prefCode, portCode, date, rng)
	if err != nil {
		return nil, false, fmt.Errorf("fetching tide data for port %s on %s: %w", portCode, date, err)
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, tideTTL) //nolint:errcheck
	}

	return data, false, nil
}
