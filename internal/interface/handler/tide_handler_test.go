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

type MockTideUsecase struct{ mock.Mock }

func (m *MockTideUsecase) GetTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error) {
	args := m.Called(ctx, prefCode, portCode, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.TideData), args.Error(1)
}

func TestTideHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tideData := &entity.TideData{PortCode: "TK", Date: "2026-04-30", TideType: "大潮"}

	tests := []struct {
		name       string
		url        string
		setupMock  func(*MockTideUsecase)
		wantStatus int
	}{
		{
			name: "正常取得",
			url:  "/api/conditions/tide?prefectureCode=13&portCode=TK&date=2026-04-30",
			setupMock: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "TK", "2026-04-30").Return(tideData, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "prefectureCodeなし → 400",
			url:        "/api/conditions/tide?portCode=TK",
			setupMock:  func(m *MockTideUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "portCodeなし → 400",
			url:        "/api/conditions/tide?prefectureCode=13",
			setupMock:  func(m *MockTideUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "prefCodeフォーマット不正 → 400",
			url:        "/api/conditions/tide?prefectureCode=ABC&portCode=TK",
			setupMock:  func(m *MockTideUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "portCodeフォーマット不正 → 400",
			url:        "/api/conditions/tide?prefectureCode=13&portCode=TK!@#",
			setupMock:  func(m *MockTideUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "日付フォーマット不正 → 400",
			url:        "/api/conditions/tide?prefectureCode=13&portCode=TK&date=20260430",
			setupMock:  func(m *MockTideUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "意味的に無効な日付 → 400",
			url:        "/api/conditions/tide?prefectureCode=13&portCode=TK&date=2026-99-99",
			setupMock:  func(m *MockTideUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "APIエラー → 500",
			url:  "/api/conditions/tide?prefectureCode=13&portCode=TK&date=2026-04-30",
			setupMock: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "TK", "2026-04-30").Return(nil, errors.New("api error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &MockTideUsecase{}
			tt.setupMock(mockUC)

			router := gin.New()
			h := NewTideHandler(mockUC)
			router.GET("/api/conditions/tide", h.Get)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			mockUC.AssertExpectations(t)
		})
	}
}
