package weather

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type MockWeatherAPI struct{ mock.Mock }

func (m *MockWeatherAPI) FetchCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.WeatherData), args.Error(1)
}

func (m *MockWeatherAPI) FetchForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.WeatherData), args.Error(1)
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
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// --- Tests ---

func TestGetCurrent_CacheMiss(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	lat, lon := 35.6762, 139.6503
	expectedData := &entity.WeatherData{
		Temperature: 22.5,
		FeelsLike:   21.0,
		WindSpeed:   3.5,
		WindDeg:     180,
		Pressure:    1013.0,
		Humidity:    60,
		Description: "晴れ",
		DateTime:    time.Now(),
	}
	expectedKey := cacheKey(lat, lon, "current")

	// キャッシュにはヒットしない
	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	// APIが呼ばれる
	mockAPI.On("FetchCurrent", ctx, lat, lon).Return(expectedData, nil)
	// キャッシュに保存される
	mockCache.On("Set", ctx, expectedKey, mock.AnythingOfType("[]uint8"), weatherTTL).Return(nil)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetCurrent(ctx, lat, lon)

	require.NoError(t, err)
	assert.Equal(t, expectedData.Temperature, result.Temperature)
	assert.Equal(t, expectedData.Description, result.Description)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetCurrent_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	lat, lon := 35.6762, 139.6503
	cachedData := &entity.WeatherData{
		Temperature: 20.0,
		Description: "曇り",
		DateTime:    time.Now(),
	}
	cachedJSON, err := json.Marshal(cachedData)
	require.NoError(t, err)

	expectedKey := cacheKey(lat, lon, "current")

	// キャッシュにヒット
	mockCache.On("Get", ctx, expectedKey).Return(cachedJSON, nil)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetCurrent(ctx, lat, lon)

	require.NoError(t, err)
	assert.Equal(t, cachedData.Temperature, result.Temperature)
	assert.Equal(t, cachedData.Description, result.Description)
	// APIは呼ばれない
	mockAPI.AssertNotCalled(t, "FetchCurrent", mock.Anything, mock.Anything, mock.Anything)
	mockCache.AssertExpectations(t)
}

func TestGetForecast_CacheMiss(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	lat, lon := 34.6937, 135.5023
	expectedData := []*entity.WeatherData{
		{Temperature: 18.0, Description: "小雨", DateTime: time.Now()},
		{Temperature: 20.0, Description: "晴れ", DateTime: time.Now().Add(3 * time.Hour)},
	}
	expectedKey := cacheKey(lat, lon, "forecast")

	// キャッシュにはヒットしない
	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	// APIが呼ばれる
	mockAPI.On("FetchForecast", ctx, lat, lon).Return(expectedData, nil)
	// キャッシュに保存される
	mockCache.On("Set", ctx, expectedKey, mock.AnythingOfType("[]uint8"), weatherTTL).Return(nil)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetForecast(ctx, lat, lon)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, expectedData[0].Temperature, result[0].Temperature)
	assert.Equal(t, expectedData[1].Description, result[1].Description)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetForecast_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	lat, lon := 34.6937, 135.5023
	cachedData := []*entity.WeatherData{
		{Temperature: 15.0, Description: "雨", DateTime: time.Now()},
	}
	cachedJSON, err := json.Marshal(cachedData)
	require.NoError(t, err)

	expectedKey := cacheKey(lat, lon, "forecast")

	// キャッシュにヒット
	mockCache.On("Get", ctx, expectedKey).Return(cachedJSON, nil)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetForecast(ctx, lat, lon)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, cachedData[0].Temperature, result[0].Temperature)
	// APIは呼ばれない
	mockAPI.AssertNotCalled(t, "FetchForecast", mock.Anything, mock.Anything, mock.Anything)
	mockCache.AssertExpectations(t)
}

func TestGetCurrent_APIError(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	lat, lon := 35.6762, 139.6503
	expectedKey := cacheKey(lat, lon, "current")
	apiErr := errors.New("weather API request failed")

	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	mockAPI.On("FetchCurrent", ctx, lat, lon).Return(nil, apiErr)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetCurrent(ctx, lat, lon)

	assert.Error(t, err)
	assert.Nil(t, result)
	// Setは呼ばれない
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetForecast_APIError(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	lat, lon := 34.6937, 135.5023
	expectedKey := cacheKey(lat, lon, "forecast")
	apiErr := errors.New("forecast API request failed")

	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	mockAPI.On("FetchForecast", ctx, lat, lon).Return(nil, apiErr)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetForecast(ctx, lat, lon)

	assert.Error(t, err)
	assert.Nil(t, result)
	// Setは呼ばれない
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCacheKey_RoundingConsistency(t *testing.T) {
	tests := []struct {
		name        string
		lat         float64
		lon         float64
		typ         string
		expectedKey string
	}{
		{
			name:        "小数4桁に丸める",
			lat:         35.676219,
			lon:         139.650318,
			typ:         "current",
			expectedKey: "weather:current:35.6762:139.6503",
		},
		{
			name:        "整数座標",
			lat:         35.0,
			lon:         139.0,
			typ:         "forecast",
			expectedKey: "weather:forecast:35.0000:139.0000",
		},
		{
			name:        "負の座標",
			lat:         -33.8688,
			lon:         151.2093,
			typ:         "current",
			expectedKey: "weather:current:-33.8688:151.2093",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cacheKey(tt.lat, tt.lon, tt.typ)
			assert.Equal(t, tt.expectedKey, key)
		})
	}
}
