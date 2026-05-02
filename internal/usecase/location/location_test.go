package location

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

type MockMapsAPI struct{ mock.Mock }

func (m *MockMapsAPI) SearchLocations(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.LocationSearchResult), args.Error(1)
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

func TestSearch_CacheMiss(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockMapsAPI{}
	mockCache := &MockCache{}

	query := "東京湾"
	limit := 5
	expectedKey := "location:東京湾:5"

	expectedResults := []*entity.LocationSearchResult{
		{
			Name:       "東京湾",
			Latitude:   35.4800,
			Longitude:  139.7300,
			Prefecture: "神奈川県",
			Region:     "関東",
		},
		{
			Name:       "東京湾奥",
			Latitude:   35.6200,
			Longitude:  139.8000,
			Prefecture: "東京都",
			Region:     "関東",
		},
	}

	// キャッシュにはヒットしない
	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	// APIが呼ばれる
	mockAPI.On("SearchLocations", ctx, query, limit).Return(expectedResults, nil)
	// キャッシュに保存される（TTL = 24h）
	mockCache.On("Set", ctx, expectedKey, mock.AnythingOfType("[]uint8"), locationCacheTTL).Return(nil)

	uc := NewLocationUsecase(mockAPI, mockCache)
	results, err := uc.Search(ctx, query, limit)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, expectedResults[0].Name, results[0].Name)
	assert.Equal(t, expectedResults[0].Latitude, results[0].Latitude)
	assert.Equal(t, expectedResults[1].Prefecture, results[1].Prefecture)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestSearch_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockMapsAPI{}
	mockCache := &MockCache{}

	query := "大阪湾"
	limit := 3
	expectedKey := "location:大阪湾:3"

	cachedResults := []*entity.LocationSearchResult{
		{
			Name:       "大阪湾",
			Latitude:   34.5500,
			Longitude:  135.3000,
			Prefecture: "大阪府",
			Region:     "近畿",
		},
	}
	cachedJSON, err := json.Marshal(cachedResults)
	require.NoError(t, err)

	// キャッシュにヒット
	mockCache.On("Get", ctx, expectedKey).Return(cachedJSON, nil)

	uc := NewLocationUsecase(mockAPI, mockCache)
	results, err := uc.Search(ctx, query, limit)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, cachedResults[0].Name, results[0].Name)
	assert.Equal(t, cachedResults[0].Latitude, results[0].Latitude)
	// APIは呼ばれない
	mockAPI.AssertNotCalled(t, "SearchLocations")
	mockCache.AssertExpectations(t)
}

func TestSearch_APIError(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockMapsAPI{}
	mockCache := &MockCache{}

	query := "不明な場所"
	limit := 10
	expectedKey := "location:不明な場所:10"
	apiErr := errors.New("maps API request failed")

	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	mockAPI.On("SearchLocations", ctx, query, limit).Return(nil, apiErr)

	uc := NewLocationUsecase(mockAPI, mockCache)
	results, err := uc.Search(ctx, query, limit)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.ErrorContains(t, err, "不明な場所")
	assert.ErrorIs(t, err, apiErr)
	// Setは呼ばれない
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}
