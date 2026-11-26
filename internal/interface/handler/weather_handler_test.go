package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWeatherUsecase struct{ mock.Mock }

func (m *MockWeatherUsecase) GetCurrent(ctx context.Context, lat, lon float64, units string) (*entity.WeatherData, bool, error) {
	args := m.Called(ctx, lat, lon, units)
	if args.Get(0) == nil {
		return nil, false, args.Error(2)
	}
	return args.Get(0).(*entity.WeatherData), args.Bool(1), args.Error(2)
}

func (m *MockWeatherUsecase) GetForecast(ctx context.Context, lat, lon float64, units string) (json.RawMessage, bool, error) {
	args := m.Called(ctx, lat, lon, units)
	if args.Get(0) == nil {
		return nil, false, args.Error(2)
	}
	return args.Get(0).(json.RawMessage), args.Bool(1), args.Error(2)
}

func serveWeather(m *MockWeatherUsecase, url string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/weather", NewWeatherHandler(m).Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	return w
}

// 応答の JSON は Next の /api/weather（FormattedWeatherData）と同じキー。
func TestWeatherHandler_Current_ResponseShape(t *testing.T) {
	gust, rain := 1.18, 3.16
	data := &entity.WeatherData{
		Lat: 44.34, Lon: 10.99, CityName: "Zocca", Country: "IT",
		Temperature: 25.33, FeelsLike: 25.59, TempMin: 24.41, TempMax: 26.9,
		Humidity: 64, Pressure: 1015, WindSpeed: 0.62, WindDeg: 349, WindGust: &gust,
		Cloudiness: 100, Visibility: 10000,
		Condition: entity.WeatherCondition{ID: 501, Main: "Rain", Description: "moderate rain", Icon: "10d"},
		Rain:      &rain,
		Sunrise:   time.Unix(1661834187, 0), Sunset: time.Unix(1661882248, 0), DateTime: time.Unix(1661870592, 0),
		Timezone: 7200,
	}
	m := &MockWeatherUsecase{}
	m.On("GetCurrent", mock.Anything, 44.34, 10.99, "metric").Return(data, false, nil)

	w := serveWeather(m, "/api/weather?lat=44.34&lon=10.99")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
		"success": true,
		"type": "current",
		"data": {
			"coordinates": {"lat": 44.34, "lon": 10.99},
			"cityName": "Zocca", "country": "IT",
			"temperature": 25.33, "feelsLike": 25.59, "tempMin": 24.41, "tempMax": 26.9,
			"humidity": 64, "pressure": 1015,
			"windSpeed": 0.62, "windDeg": 349, "windGust": 1.18,
			"cloudiness": 100, "visibility": 10000,
			"weather": {"id": 501, "main": "Rain", "description": "moderate rain", "icon": "10d"},
			"rain": 3.16,
			"sunrise": "2022-08-30T04:36:27.000Z",
			"sunset": "2022-08-30T17:57:28.000Z",
			"dataTime": "2022-08-30T14:43:12.000Z",
			"timezone": 7200
		},
		"fromCache": false,
		"fetchedAt": "IGNORED"
	}`, replaceFetchedAt(w.Body.String()))
	m.AssertExpectations(t)
}

// 突風・雨・雪が無ければ、Next（undefined は JSON に出ない）と同じくキーごと出さない。
func TestWeatherHandler_Current_OmitsAbsentOptionals(t *testing.T) {
	m := &MockWeatherUsecase{}
	m.On("GetCurrent", mock.Anything, 35.0, 139.0, "imperial").Return(&entity.WeatherData{}, true, nil)

	w := serveWeather(m, "/api/weather?lat=35&lon=139&units=imperial")

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		FromCache bool           `json:"fromCache"`
		Data      map[string]any `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.FromCache)
	assert.NotContains(t, body.Data, "windGust")
	assert.NotContains(t, body.Data, "rain")
	assert.NotContains(t, body.Data, "snow")
}

// 予報は上流の応答をそのまま data に入れる（Next と同じ）。
func TestWeatherHandler_Forecast(t *testing.T) {
	raw := json.RawMessage(`{"cod":"200","cnt":1,"list":[{"dt":1}]}`)
	m := &MockWeatherUsecase{}
	m.On("GetForecast", mock.Anything, 35.0, 139.0, "standard").Return(raw, true, nil)

	w := serveWeather(m, "/api/weather?lat=35&lon=139&type=forecast&units=standard")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"success":true,"type":"forecast","data":{"cod":"200","cnt":1,"list":[{"dt":1}]},"fromCache":true,"fetchedAt":"IGNORED"}`,
		replaceFetchedAt(w.Body.String()))
}

func TestWeatherHandler_Errors(t *testing.T) {
	invalid := domain.NewInvalidUpstreamResponse("OpenWeatherMap returned an invalid wind.speed")
	tests := []struct {
		name       string
		url        string
		setup      func(m *MockWeatherUsecase)
		wantStatus int
		wantBody   string
	}{
		{"座標が不正", "/api/weather?lat=999&lon=139", nil, 400, ""},
		{
			"type が不正", "/api/weather?lat=35&lon=139&type=hourly", nil, 400,
			`{"error":"typeは \"current\" または \"forecast\" である必要があります","code":"INVALID_PARAMS","status":400}`,
		},
		{
			"units が不正", "/api/weather?lat=35&lon=139&units=kelvin", nil, 400,
			`{"error":"unitsは \"standard\", \"metric\", または \"imperial\" である必要があります","code":"INVALID_PARAMS","status":400}`,
		},
		{"風速・気圧が不正なら 502", "/api/weather?lat=35&lon=139", func(m *MockWeatherUsecase) {
			m.On("GetCurrent", mock.Anything, 35.0, 139.0, "metric").Return(nil, false, invalid)
		}, 502, `{"success":false,"error":"サーバーで問題が発生しました。サポートに連絡してください。","code":"SERVER_ERROR","retryable":true}`},
		{"予報のタイムアウトは 504", "/api/weather?lat=35&lon=139&type=forecast", func(m *MockWeatherUsecase) {
			m.On("GetForecast", mock.Anything, 35.0, 139.0, "metric").
				Return(nil, false, &domain.UpstreamError{Kind: domain.UpstreamTimeout, Retryable: true})
		}, 504, `{"success":false,"error":"通信がタイムアウトしました。インターネット接続を確認して、もう一度お試しください。","code":"TIMEOUT","retryable":true}`},
		{"想定外は 500", "/api/weather?lat=35&lon=139", func(m *MockWeatherUsecase) {
			m.On("GetCurrent", mock.Anything, 35.0, 139.0, "metric").Return(nil, false, errors.New("boom"))
		}, 500, `{"success":false,"error":"天気データの取得中にエラーが発生しました","code":"INTERNAL_ERROR"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MockWeatherUsecase{}
			if tt.setup != nil {
				tt.setup(m)
			}

			w := serveWeather(m, tt.url)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, w.Body.String())
			}
			m.AssertExpectations(t)
		})
	}
}
