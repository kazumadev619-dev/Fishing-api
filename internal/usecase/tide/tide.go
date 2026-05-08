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
	FetchTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error)
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

// GetTideData は指定された港コード・日付の潮汐データを返す。
// キャッシュにデータがある場合はAPIを呼ばずキャッシュから返す。
func (u *TideUsecase) GetTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error) {
	key := fmt.Sprintf("tide:%s:%s:%s", prefCode, portCode, date)

	cached, _ := u.cache.Get(ctx, key) //nolint:errcheck // cache miss is acceptable
	if cached != nil {
		var data entity.TideData
		if err := json.Unmarshal(cached, &data); err == nil {
			return &data, nil
		}
	}

	data, err := u.api.FetchTideData(ctx, prefCode, portCode, date)
	if err != nil {
		return nil, fmt.Errorf("fetching tide data for port %s on %s: %w", portCode, date, err)
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, tideTTL) //nolint:errcheck
	}

	return data, nil
}
