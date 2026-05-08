package tide

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

type MockTideAPI struct{ mock.Mock }

func (m *MockTideAPI) FetchTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error) {
	args := m.Called(ctx, prefCode, portCode, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.TideData), args.Error(1)
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

func TestGetTideData_CacheMiss(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockTideAPI{}
	mockCache := &MockCache{}

	prefCode := "14"
	portCode := "TK"
	date := "20260430"

	expectedData := &entity.TideData{
		PortCode: portCode,
		Date:     date,
		HighTides: []entity.TideEvent{
			{Time: time.Now(), Height: 180.0},
		},
		LowTides: []entity.TideEvent{
			{Time: time.Now().Add(6 * time.Hour), Height: 20.0},
		},
		TideType: "大潮",
	}
	expectedKey := "tide:" + prefCode + ":" + portCode + ":" + date

	// キャッシュにはヒットしない
	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	// APIが呼ばれる
	mockAPI.On("FetchTideData", ctx, prefCode, portCode, date).Return(expectedData, nil)
	// キャッシュに保存される（TTL = 6h）
	mockCache.On("Set", ctx, expectedKey, mock.AnythingOfType("[]uint8"), tideTTL).Return(nil)

	uc := NewTideUsecase(mockAPI, mockCache)
	result, err := uc.GetTideData(ctx, prefCode, portCode, date)

	require.NoError(t, err)
	assert.Equal(t, expectedData.PortCode, result.PortCode)
	assert.Equal(t, expectedData.Date, result.Date)
	assert.Equal(t, expectedData.TideType, result.TideType)
	assert.Len(t, result.HighTides, 1)
	assert.Len(t, result.LowTides, 1)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetTideData_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockTideAPI{}
	mockCache := &MockCache{}

	prefCode := "14"
	portCode := "TK"
	date := "20260430"

	cachedData := &entity.TideData{
		PortCode: portCode,
		Date:     date,
		HighTides: []entity.TideEvent{
			{Time: time.Now(), Height: 200.0},
		},
		LowTides: []entity.TideEvent{},
		TideType: "中潮",
	}
	cachedJSON, err := json.Marshal(cachedData)
	require.NoError(t, err)

	expectedKey := "tide:" + prefCode + ":" + portCode + ":" + date

	// キャッシュにヒット
	mockCache.On("Get", ctx, expectedKey).Return(cachedJSON, nil)

	uc := NewTideUsecase(mockAPI, mockCache)
	result, err := uc.GetTideData(ctx, prefCode, portCode, date)

	require.NoError(t, err)
	assert.Equal(t, cachedData.PortCode, result.PortCode)
	assert.Equal(t, cachedData.TideType, result.TideType)
	assert.Len(t, result.HighTides, 1)
	// APIは呼ばれない
	mockAPI.AssertNotCalled(t, "FetchTideData")
	mockCache.AssertExpectations(t)
}

func TestGetTideData_APIError(t *testing.T) {
	ctx := context.Background()
	mockAPI := &MockTideAPI{}
	mockCache := &MockCache{}

	prefCode := "14"
	portCode := "TK"
	date := "20260430"

	expectedKey := "tide:" + prefCode + ":" + portCode + ":" + date
	apiErr := errors.New("tide API request failed")

	mockCache.On("Get", ctx, expectedKey).Return(nil, errors.New("cache miss"))
	mockAPI.On("FetchTideData", ctx, prefCode, portCode, date).Return(nil, apiErr)

	uc := NewTideUsecase(mockAPI, mockCache)
	result, err := uc.GetTideData(ctx, prefCode, portCode, date)

	assert.Error(t, err)
	assert.Nil(t, result)
	// Setは呼ばれない
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}
