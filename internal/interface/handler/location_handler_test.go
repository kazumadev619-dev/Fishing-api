package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLocationUsecase struct{ mock.Mock }

func (m *MockLocationUsecase) Search(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.LocationSearchResult), args.Error(1)
}

func TestLocationHandler_Search(t *testing.T) {
	gin.SetMode(gin.TestMode)

	results := []*entity.LocationSearchResult{
		{Name: "東京湾", Latitude: 35.5, Longitude: 139.8, Prefecture: "東京都"},
	}

	tests := []struct {
		name       string
		url        string
		setupMock  func(*MockLocationUsecase)
		wantStatus int
	}{
		{
			name: "正常検索",
			url:  "/api/locations/search?q=東京湾",
			setupMock: func(m *MockLocationUsecase) {
				m.On("Search", mock.Anything, "東京湾", 5).Return(results, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "limit指定",
			url:  "/api/locations/search?q=東京湾&limit=3",
			setupMock: func(m *MockLocationUsecase) {
				m.On("Search", mock.Anything, "東京湾", 3).Return(results, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "qなし → 400",
			url:        "/api/locations/search",
			setupMock:  func(m *MockLocationUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "q短すぎ → 400",
			url:        "/api/locations/search?q=a",
			setupMock:  func(m *MockLocationUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit=0 → 400",
			url:        "/api/locations/search?q=東京湾&limit=0",
			setupMock:  func(m *MockLocationUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "limit不正 → 400",
			url:        "/api/locations/search?q=東京湾&limit=abc",
			setupMock:  func(m *MockLocationUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "検索エラー → 500",
			url:  "/api/locations/search?q=東京湾",
			setupMock: func(m *MockLocationUsecase) {
				m.On("Search", mock.Anything, "東京湾", 5).Return(nil, errors.New("maps error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &MockLocationUsecase{}
			tt.setupMock(mockUC)

			router := gin.New()
			h := NewLocationHandler(mockUC)
			router.GET("/api/locations/search", h.Search)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			mockUC.AssertExpectations(t)
		})
	}
}
