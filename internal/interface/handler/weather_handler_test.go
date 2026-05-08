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

type MockWeatherUsecase struct{ mock.Mock }

func (m *MockWeatherUsecase) GetCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.WeatherData), args.Error(1)
}

func (m *MockWeatherUsecase) GetForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.WeatherData), args.Error(1)
}

func TestWeatherHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	weatherData := &entity.WeatherData{Temperature: 20.0, WindSpeed: 5.0}
	forecastData := []*entity.WeatherData{{Temperature: 18.0}, {Temperature: 22.0}}

	tests := []struct {
		name       string
		url        string
		setupMock  func(*MockWeatherUsecase)
		wantStatus int
	}{
		{
			name: "current 正常取得",
			url:  "/api/weather?lat=35.6895&lon=139.6917&type=current",
			setupMock: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.6895, 139.6917).Return(weatherData, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "current デフォルト（typeなし）",
			url:  "/api/weather?lat=35.6895&lon=139.6917",
			setupMock: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.6895, 139.6917).Return(weatherData, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "forecast 正常取得",
			url:  "/api/weather?lat=35.6895&lon=139.6917&type=forecast",
			setupMock: func(m *MockWeatherUsecase) {
				m.On("GetForecast", mock.Anything, 35.6895, 139.6917).Return(forecastData, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "latなし → 400",
			url:        "/api/weather?lon=139.6917",
			setupMock:  func(m *MockWeatherUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid lat → 400",
			url:        "/api/weather?lat=invalid&lon=139.6917",
			setupMock:  func(m *MockWeatherUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "lat範囲外 → 400",
			url:        "/api/weather?lat=200&lon=139.6917",
			setupMock:  func(m *MockWeatherUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "不明なtype → 400",
			url:        "/api/weather?lat=35.6895&lon=139.6917&type=unknown",
			setupMock:  func(m *MockWeatherUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "current APIエラー → 500",
			url:  "/api/weather?lat=35.6895&lon=139.6917&type=current",
			setupMock: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.6895, 139.6917).Return(nil, errors.New("api error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "forecast APIエラー → 500",
			url:  "/api/weather?lat=35.6895&lon=139.6917&type=forecast",
			setupMock: func(m *MockWeatherUsecase) {
				m.On("GetForecast", mock.Anything, 35.6895, 139.6917).Return(nil, errors.New("api error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &MockWeatherUsecase{}
			tt.setupMock(mockUC)

			router := gin.New()
			h := NewWeatherHandler(mockUC)
			router.GET("/api/weather", h.Get)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			mockUC.AssertExpectations(t)
		})
	}
}
