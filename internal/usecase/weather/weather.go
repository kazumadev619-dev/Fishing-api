package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/pkg/validator"
)

// WeatherAPI は外部天気APIクライアントのインターフェース。
// infrastructure/external/ の実装が満たすこと。units は standard / metric / imperial。
type WeatherAPI interface {
	FetchCurrent(ctx context.Context, lat, lon float64, units string) (*entity.WeatherData, error)
	FetchForecast(ctx context.Context, lat, lon float64, units string) (json.RawMessage, error)
}

// Cache はキャッシュストアのインターフェース。
// infrastructure/cache/ の実装が満たすこと。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

const weatherTTL = 30 * time.Minute

// WeatherUsecase は天気情報の取得ビジネスロジックを担う。
// Redisキャッシュを使い、キャッシュミス時のみ外部APIを呼び出す。取得に失敗したときはキャッシュに書かない。
type WeatherUsecase struct {
	api   WeatherAPI
	cache Cache
}

// NewWeatherUsecase は WeatherUsecase を生成する。
func NewWeatherUsecase(api WeatherAPI, cache Cache) *WeatherUsecase {
	return &WeatherUsecase{api: api, cache: cache}
}

// GetCurrent は指定座標の現在天気と、キャッシュから返したかを返す。
func (u *WeatherUsecase) GetCurrent(ctx context.Context, lat, lon float64, units string) (*entity.WeatherData, bool, error) {
	key := cacheKey(lat, lon, "current", units)

	cached, _ := u.cache.Get(ctx, key) //nolint:errcheck // cache miss is acceptable
	if cached != nil {
		var data entity.WeatherData
		if err := json.Unmarshal(cached, &data); err == nil {
			return &data, true, nil
		}
	}

	data, err := u.api.FetchCurrent(ctx, lat, lon, units)
	if err != nil {
		return nil, false, fmt.Errorf("fetching current weather for %.4f,%.4f: %w", lat, lon, err)
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, weatherTTL) //nolint:errcheck
	}

	return data, false, nil
}

// GetForecast は指定座標の天気予報（上流の応答のまま）と、キャッシュから返したかを返す。
func (u *WeatherUsecase) GetForecast(ctx context.Context, lat, lon float64, units string) (json.RawMessage, bool, error) {
	key := cacheKey(lat, lon, "forecast", units)

	cached, _ := u.cache.Get(ctx, key) //nolint:errcheck // cache miss is acceptable
	if cached != nil && json.Valid(cached) {
		return json.RawMessage(cached), true, nil
	}

	data, err := u.api.FetchForecast(ctx, lat, lon, units)
	if err != nil {
		return nil, false, fmt.Errorf("fetching weather forecast for %.4f,%.4f: %w", lat, lon, err)
	}

	u.cache.Set(ctx, key, data, weatherTTL) //nolint:errcheck

	return data, false, nil
}

// cacheKey は座標・種別・単位系からキャッシュキーを生成する。
// 座標は小数4桁に丸めて同一エリアのリクエストを同一キーに集約する。
func cacheKey(lat, lon float64, typ, units string) string {
	roundedLat := validator.RoundCoordinate(lat, 4)
	roundedLon := validator.RoundCoordinate(lon, 4)
	return fmt.Sprintf("weather:%s:%s:%.4f:%.4f", typ, units, roundedLat, roundedLon)
}
