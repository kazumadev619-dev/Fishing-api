package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestOpenAPI_ResponseConformance は api/openapi.yaml の仕様と実際の HTTP 応答（ステータス・ヘッダー・本文）を照合する。
func TestOpenAPI_ResponseConformance(t *testing.T) {
	ctx := context.Background()
	specPath := filepath.Join("..", "..", "..", "api", "openapi.yaml")

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(specPath)
	require.NoError(t, err)
	require.NoError(t, doc.Validate(ctx))

	specRouter, err := legacy.NewRouter(doc)
	require.NoError(t, err)

	validate := func(t *testing.T, req *http.Request, w *httptest.ResponseRecorder, wantStatus int) {
		t.Helper()
		require.Equal(t, wantStatus, w.Code, "body: %s", w.Body.String())

		route, pathParams, err := specRouter.FindRoute(req)
		require.NoError(t, err)

		reqInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}
		respInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: reqInput,
			Status:                 w.Code,
			Header:                 w.Header(),
			Options: &openapi3filter.Options{
				IncludeResponseStatus: true,
			},
		}
		respInput.SetBodyBytes(w.Body.Bytes())

		assert.NoError(t, openapi3filter.ValidateResponse(ctx, respInput), "body: %s", w.Body.String())
	}

	gust, rain := 1.18, 3.16
	sampleWeather := &entity.WeatherData{
		Lat: 35.68, Lon: 139.69, CityName: "Tokyo", Country: "JP",
		Temperature: 20.5, FeelsLike: 20.0, TempMin: 18.0, TempMax: 22.0,
		Humidity: 60, Pressure: 1013, WindSpeed: 3.2, WindDeg: 180, WindGust: &gust,
		Cloudiness: 20, Visibility: 10000,
		Condition: entity.WeatherCondition{ID: 801, Main: "Clouds", Description: "few clouds", Icon: "02d"},
		Rain:      &rain,
		Sunrise:   time.Unix(1700000000, 0), Sunset: time.Unix(1700040000, 0), DateTime: time.Unix(1700020000, 0),
		Timezone: 32400,
	}

	moonName := "満月"
	sampleTide := &entity.TideData{
		Port: entity.TidePort{PrefectureCode: "13", PortCode: "2", PortName: "芝浦"},
		Days: []entity.DailyTide{{
			Date:  "2026-11-27",
			Sun:   entity.SunInfo{AstroTwilight: []string{"04:58", "17:59"}, RegularTwilight: []string{"06:00", "16:57"}, Rise: "06:28", Midline: "11:29", Set: "16:29"},
			Moon:  entity.MoonInfo{Brightness: "90.9", Age: "17.8", Title: "中潮", Rise: "26日 17:46", Midline: "27日 01:34", Set: "27日 09:19", Name: &moonName},
			Tide:  []entity.TideEvent{{Time: "00:00", UnixMs: 1795705200000, CM: -13.6}},
			Flood: []entity.TideEvent{{Time: "07:26", UnixMs: 1795731960000, CM: 197.7}},
			Ebb:   []entity.TideEvent{{Time: "00:13", UnixMs: 1795705980000, CM: -14.2}},
		}},
		DataTime: time.Date(2026, 11, 27, 1, 2, 3, 4_000_000, time.UTC),
	}

	sampleTideNullMoon := &entity.TideData{
		Port: entity.TidePort{PrefectureCode: "13", PortCode: "2", PortName: "芝浦"},
		Days: []entity.DailyTide{{
			Date:  "2026-11-28",
			Sun:   entity.SunInfo{AstroTwilight: []string{}, RegularTwilight: []string{}, Rise: "06:29", Midline: "11:29", Set: "16:29"},
			Moon:  entity.MoonInfo{Brightness: "80.0", Age: "18.8", Title: "中潮", Rise: "27日 18:46", Midline: "28日 02:34", Set: "28日 10:19", Name: nil},
			Tide:  []entity.TideEvent{},
			Flood: []entity.TideEvent{},
			Ebb:   []entity.TideEvent{},
		}},
		DataTime: time.Date(2026, 11, 28, 1, 2, 3, 4_000_000, time.UTC),
	}

	tests := []struct {
		name          string
		url           string
		apiConfigured bool
		setupWeather  func(m *MockWeatherUsecase)
		setupTide     func(m *MockTideUsecase)
		wantStatus    int
	}{
		{
			name:          "weather current 200",
			url:           "/api/weather?lat=35.68&lon=139.69",
			apiConfigured: true,
			setupWeather: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.68, 139.69, "metric").Return(sampleWeather, false, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "weather forecast 200",
			url:           "/api/weather?lat=35.68&lon=139.69&type=forecast",
			apiConfigured: true,
			setupWeather: func(m *MockWeatherUsecase) {
				m.On("GetForecast", mock.Anything, 35.68, 139.69, "metric").
					Return(json.RawMessage(`{"cod":"200","cnt":1,"list":[{"dt":1700020000}]}`), true, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "weather 400 (range error)",
			url:           "/api/weather?lat=999&lon=139.69",
			apiConfigured: true,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "weather 400 (wrapper parse error)",
			url:           "/api/weather?lat=abc&lon=139.69",
			apiConfigured: true,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "weather 500",
			url:           "/api/weather?lat=35.68&lon=139.69",
			apiConfigured: true,
			setupWeather: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.68, 139.69, "metric").Return(nil, false, errors.New("boom"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:          "weather 502",
			url:           "/api/weather?lat=35.68&lon=139.69",
			apiConfigured: true,
			setupWeather: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.68, 139.69, "metric").
					Return(nil, false, &domain.UpstreamError{Kind: domain.UpstreamServerError, StatusCode: 503, Retryable: true})
			},
			wantStatus: http.StatusBadGateway,
		},
		{
			name:          "weather 503 (API_NOT_CONFIGURED)",
			url:           "/api/weather?lat=35.68&lon=139.69",
			apiConfigured: false,
			wantStatus:    http.StatusServiceUnavailable,
		},
		{
			name:          "weather 503 (DB_UNAVAILABLE)",
			url:           "/api/weather?lat=35.68&lon=139.69",
			apiConfigured: true,
			setupWeather: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.68, 139.69, "metric").Return(nil, false, domain.ErrDBUnavailable)
			},
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:          "weather 504",
			url:           "/api/weather?lat=35.68&lon=139.69",
			apiConfigured: true,
			setupWeather: func(m *MockWeatherUsecase) {
				m.On("GetCurrent", mock.Anything, 35.68, 139.69, "metric").
					Return(nil, false, &domain.UpstreamError{Kind: domain.UpstreamTimeout, Retryable: true})
			},
			wantStatus: http.StatusGatewayTimeout,
		},
		{
			name:          "tide 200 (cached)",
			url:           "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27&range=day",
			apiConfigured: true,
			setupTide: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "2", "2026-11-27", "day").Return(sampleTide, true, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "tide 200 (fresh, moon name null)",
			url:           "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-28&range=week",
			apiConfigured: true,
			setupTide: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "2", "2026-11-28", "week").Return(sampleTideNullMoon, false, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "tide 400 (wrapper missing param)",
			url:           "/api/conditions/tide?portCode=2",
			apiConfigured: true,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "tide 400 (invalid date)",
			url:           "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-02-30",
			apiConfigured: true,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:          "tide 500",
			url:           "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27",
			apiConfigured: true,
			setupTide: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "2", "2026-11-27", "week").Return(nil, false, errors.New("boom"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:          "tide 502",
			url:           "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27",
			apiConfigured: true,
			setupTide: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "2", "2026-11-27", "week").
					Return(nil, false, &domain.UpstreamError{Kind: domain.UpstreamClientError, StatusCode: 400, Retryable: false})
			},
			wantStatus: http.StatusBadGateway,
		},
		{
			name:          "tide 503",
			url:           "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27",
			apiConfigured: true,
			setupTide: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "2", "2026-11-27", "week").Return(nil, false, domain.ErrDBUnavailable)
			},
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:          "tide 504",
			url:           "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27",
			apiConfigured: true,
			setupTide: func(m *MockTideUsecase) {
				m.On("GetTideData", mock.Anything, "13", "2", "2026-11-27", "week").
					Return(nil, false, &domain.UpstreamError{Kind: domain.UpstreamTimeout, Retryable: true})
			},
			wantStatus: http.StatusGatewayTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mw := &MockWeatherUsecase{}
			mt := &MockTideUsecase{}
			if tt.setupWeather != nil {
				tt.setupWeather(mw)
			}
			if tt.setupTide != nil {
				tt.setupTide(mt)
			}

			r := gin.New()
			RegisterOpenAPI(r, NewWeatherHandler(mw, tt.apiConfigured), NewTideHandler(mt))

			req := httptest.NewRequestWithContext(ctx, http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			validate(t, req, w, tt.wantStatus)
			mw.AssertExpectations(t)
			mt.AssertExpectations(t)
		})
	}
}
