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
// infrastructure/external/ の実装が満たすこと。
type WeatherAPI interface {
	FetchCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error)
	FetchForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error)
}

// Cache はキャッシュストアのインターフェース。
// infrastructure/cache/ の実装が満たすこと。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

const weatherTTL = 30 * time.Minute

// WeatherUsecase は天気情報の取得ビジネスロジックを担う。
// Redisキャッシュを使い、キャッシュミス時のみ外部APIを呼び出す。
type WeatherUsecase struct {
	api   WeatherAPI
	cache Cache
}

// NewWeatherUsecase は WeatherUsecase を生成する。
func NewWeatherUsecase(api WeatherAPI, cache Cache) *WeatherUsecase {
	return &WeatherUsecase{api: api, cache: cache}
}

// GetCurrent は指定座標の現在天気を返す。
// キャッシュにデータがある場合はAPIを呼ばずキャッシュから返す。
func (u *WeatherUsecase) GetCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error) {
	key := cacheKey(lat, lon, "current")

	if cached, _ := u.cache.Get(ctx, key); cached != nil {
		var data entity.WeatherData
		if err := json.Unmarshal(cached, &data); err == nil {
			return &data, nil
		}
	}

	data, err := u.api.FetchCurrent(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, weatherTTL) //nolint:errcheck
	}

	return data, nil
}

// GetForecast は指定座標の天気予報リストを返す。
// キャッシュにデータがある場合はAPIを呼ばずキャッシュから返す。
func (u *WeatherUsecase) GetForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error) {
	key := cacheKey(lat, lon, "forecast")

	if cached, _ := u.cache.Get(ctx, key); cached != nil {
		var data []*entity.WeatherData
		if err := json.Unmarshal(cached, &data); err == nil {
			return data, nil
		}
	}

	data, err := u.api.FetchForecast(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, weatherTTL) //nolint:errcheck
	}

	return data, nil
}

// cacheKey は座標と種別からキャッシュキーを生成する。
// 座標は小数4桁に丸めて同一エリアのリクエストを同一キーに集約する。
func cacheKey(lat, lon float64, typ string) string {
	roundedLat := validator.RoundCoordinate(lat, 4)
	roundedLon := validator.RoundCoordinate(lon, 4)
	return fmt.Sprintf("weather:%s:%.4f:%.4f", typ, roundedLat, roundedLon)
}
