package tide

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

type MockTideAPI struct{ mock.Mock }

func (m *MockTideAPI) FetchTideData(ctx context.Context, prefCode, portCode, date, tideRange string) (*entity.TideData, error) {
	args := m.Called(ctx, prefCode, portCode, date, tideRange)
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
	return m.Called(ctx, key, value, ttl).Error(0)
}

const key = "tide:13:2:2026-11-27:week"

func sample() *entity.TideData {
	return &entity.TideData{
		Port: entity.TidePort{PrefectureCode: "13", PortCode: "2", PortName: "芝浦"},
		Days: []entity.DailyTide{{
			Date:  "2026-11-27",
			Moon:  entity.MoonInfo{Title: "中潮"},
			Flood: []entity.TideEvent{{Time: "07:26", UnixMs: 1795731960000, CM: 197.7}},
		}},
		DataTime: time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC),
	}
}

func TestGetTideData_CacheMiss(t *testing.T) {
	api, cache := &MockTideAPI{}, &MockCache{}
	cache.On("Get", mock.Anything, key).Return(nil, nil)
	api.On("FetchTideData", mock.Anything, "13", "2", "2026-11-27", "week").Return(sample(), nil)
	cache.On("Set", mock.Anything, key, mock.Anything, tideTTL).Return(nil)

	got, fromCache, err := NewTideUsecase(api, cache).GetTideData(t.Context(), "13", "2", "2026-11-27", "week")

	require.NoError(t, err)
	assert.False(t, fromCache)
	assert.Equal(t, sample(), got)
	api.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestGetTideData_CacheHit(t *testing.T) {
	api, cache := &MockTideAPI{}, &MockCache{}
	b, err := json.Marshal(sample())
	require.NoError(t, err)
	cache.On("Get", mock.Anything, key).Return(b, nil)

	got, fromCache, err := NewTideUsecase(api, cache).GetTideData(t.Context(), "13", "2", "2026-11-27", "week")

	require.NoError(t, err)
	assert.True(t, fromCache)
	assert.Equal(t, sample(), got)
	api.AssertNotCalled(t, "FetchTideData", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// 上流の失敗は分類を保ったまま返し、キャッシュに書かない。
func TestGetTideData_APIError_NotCached(t *testing.T) {
	api, cache := &MockTideAPI{}, &MockCache{}
	cache.On("Get", mock.Anything, key).Return(nil, nil)
	upstream := domain.NewInvalidUpstreamResponse("status 0")
	api.On("FetchTideData", mock.Anything, "13", "2", "2026-11-27", "week").Return(nil, upstream)

	_, _, err := NewTideUsecase(api, cache).GetTideData(t.Context(), "13", "2", "2026-11-27", "week")

	var ue *domain.UpstreamError
	require.True(t, errors.As(err, &ue))
	cache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
