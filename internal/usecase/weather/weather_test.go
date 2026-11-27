package weather

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type MockWeatherAPI struct{ mock.Mock }

func (m *MockWeatherAPI) FetchCurrent(ctx context.Context, lat, lon float64, units string) (*entity.WeatherData, error) {
	args := m.Called(ctx, lat, lon, units)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.WeatherData), args.Error(1)
}

func (m *MockWeatherAPI) FetchForecast(ctx context.Context, lat, lon float64, units string) (json.RawMessage, error) {
	args := m.Called(ctx, lat, lon, units)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(json.RawMessage), args.Error(1)
}

type MockCache struct{ mock.Mock }

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return m.Called(ctx, key, value, ttl).Error(0)
}

// --- Tests ---

const lat, lon = 35.6762, 139.6503

func sample() *entity.WeatherData {
	return &entity.WeatherData{
		Temperature: 22.5, WindSpeed: 3.2, Pressure: 1013,
		Sunrise:  time.Unix(1661834187, 0).UTC(),
		DateTime: time.Unix(1661870592, 0).UTC(),
	}
}

func TestGetCurrent_CacheMiss(t *testing.T) {
	api, cache := &MockWeatherAPI{}, &MockCache{}
	key := "weather:current:metric:35.6762:139.6503"
	cache.On("Get", mock.Anything, key).Return(nil, nil)
	api.On("FetchCurrent", mock.Anything, lat, lon, "metric").Return(sample(), nil)
	cache.On("Set", mock.Anything, key, mock.Anything, weatherTTL).Return(nil)

	got, fromCache, err := NewWeatherUsecase(api, cache).GetCurrent(t.Context(), lat, lon, "metric")

	require.NoError(t, err)
	assert.False(t, fromCache)
	assert.Equal(t, sample(), got)
	api.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestGetCurrent_CacheHit(t *testing.T) {
	api, cache := &MockWeatherAPI{}, &MockCache{}
	b, err := json.Marshal(sample())
	require.NoError(t, err)
	cache.On("Get", mock.Anything, "weather:current:imperial:35.6762:139.6503").Return(b, nil)

	got, fromCache, err := NewWeatherUsecase(api, cache).GetCurrent(t.Context(), lat, lon, "imperial")

	require.NoError(t, err)
	assert.True(t, fromCache)
	assert.Equal(t, sample(), got)
	api.AssertNotCalled(t, "FetchCurrent", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	api.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestGetForecast_CacheMissThenHit(t *testing.T) {
	api, cache := &MockWeatherAPI{}, &MockCache{}
	key := "weather:forecast:metric:35.6762:139.6503"
	raw := json.RawMessage(`{"cod":"200","list":[]}`)
	cache.On("Get", mock.Anything, key).Return(nil, nil).Once()
	api.On("FetchForecast", mock.Anything, lat, lon, "metric").Return(raw, nil).Once()
	cache.On("Set", mock.Anything, key, []byte(raw), weatherTTL).Return(nil)

	got, fromCache, err := NewWeatherUsecase(api, cache).GetForecast(t.Context(), lat, lon, "metric")
	require.NoError(t, err)
	assert.False(t, fromCache)
	assert.JSONEq(t, string(raw), string(got))

	cache.On("Get", mock.Anything, key).Return([]byte(raw), nil).Once()
	got, fromCache, err = NewWeatherUsecase(api, cache).GetForecast(t.Context(), lat, lon, "metric")
	require.NoError(t, err)
	assert.True(t, fromCache)
	assert.JSONEq(t, string(raw), string(got))
	api.AssertExpectations(t)
}

// 不正な値（風速・気圧）で弾かれた応答や上流の失敗は、キャッシュに書かない（Next #414）。
func TestWeather_UpstreamError_NotCached(t *testing.T) {
	invalid := domain.NewInvalidUpstreamResponse("OpenWeatherMap returned an invalid wind.speed")
	tests := []struct {
		name string
		call func(u *WeatherUsecase, api *MockWeatherAPI) error
	}{
		{"current", func(u *WeatherUsecase, api *MockWeatherAPI) error {
			api.On("FetchCurrent", mock.Anything, lat, lon, "metric").Return(nil, invalid)
			_, _, err := u.GetCurrent(t.Context(), lat, lon, "metric")
			return err
		}},
		{"forecast", func(u *WeatherUsecase, api *MockWeatherAPI) error {
			api.On("FetchForecast", mock.Anything, lat, lon, "metric").Return(nil, invalid)
			_, _, err := u.GetForecast(t.Context(), lat, lon, "metric")
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, cache := &MockWeatherAPI{}, &MockCache{}
			cache.On("Get", mock.Anything, mock.Anything).Return(nil, nil)

			err := tt.call(NewWeatherUsecase(api, cache), api)

			var ue *domain.UpstreamError
			require.True(t, errors.As(err, &ue))
			cache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			api.AssertExpectations(t)
			cache.AssertExpectations(t)
		})
	}
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		lat, lon float64
		typ      string
		units    string
		want     string
	}{
		{"小数4桁に丸める", 35.676219, 139.650318, "current", "metric", "weather:current:metric:35.6762:139.6503"},
		{"整数座標", 35.0, 139.0, "forecast", "standard", "weather:forecast:standard:35.0000:139.0000"},
		{"負の座標", -33.8688, 151.2093, "current", "imperial", "weather:current:imperial:-33.8688:151.2093"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cacheKey(tt.lat, tt.lon, tt.typ, tt.units))
		})
	}
}
