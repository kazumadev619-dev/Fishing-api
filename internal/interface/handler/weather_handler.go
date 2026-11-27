package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/pkg/validator"
)

type WeatherFetcher interface {
	GetCurrent(ctx context.Context, lat, lon float64, units string) (*entity.WeatherData, bool, error)
	GetForecast(ctx context.Context, lat, lon float64, units string) (json.RawMessage, bool, error)
}

type WeatherHandler struct {
	usecase       WeatherFetcher
	apiConfigured bool // OpenWeatherMap のキーが設定されているか
}

func NewWeatherHandler(uc WeatherFetcher, apiConfigured bool) *WeatherHandler {
	return &WeatherHandler{usecase: uc, apiConfigured: apiConfigured}
}

var weatherUnits = []string{"standard", "metric", "imperial"}

// 応答の形は Next の /api/weather（src/lib/openWeatherService.ts の FormattedWeatherData）と同じ。
type weatherResponse struct {
	Success   bool   `json:"success"`
	Type      string `json:"type"`
	Data      any    `json:"data"`
	FromCache bool   `json:"fromCache"`
	FetchedAt string `json:"fetchedAt"`
}

type weatherBody struct {
	Coordinates struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"coordinates"`
	CityName    string   `json:"cityName"`
	Country     string   `json:"country"`
	Temperature float64  `json:"temperature"`
	FeelsLike   float64  `json:"feelsLike"`
	TempMin     float64  `json:"tempMin"`
	TempMax     float64  `json:"tempMax"`
	Humidity    int      `json:"humidity"`
	Pressure    float64  `json:"pressure"`
	WindSpeed   float64  `json:"windSpeed"`
	WindDeg     int      `json:"windDeg"`
	WindGust    *float64 `json:"windGust,omitempty"`
	Cloudiness  int      `json:"cloudiness"`
	Visibility  int      `json:"visibility"`
	Weather     struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Rain     *float64 `json:"rain,omitempty"`
	Snow     *float64 `json:"snow,omitempty"`
	Sunrise  string   `json:"sunrise"`
	Sunset   string   `json:"sunset"`
	DataTime string   `json:"dataTime"`
	Timezone int      `json:"timezone"`
}

func badWeatherRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg, "code": "INVALID_PARAMS", "status": http.StatusBadRequest})
}

func (h *WeatherHandler) Get(c *gin.Context) {
	// Next と同じく、クエリの検証とキャッシュより前に確かめる。
	if !h.apiConfigured {
		slog.ErrorContext(c.Request.Context(), "OPENWEATHER_API_KEY is not configured")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Weather API is not configured", "code": "API_NOT_CONFIGURED", "status": http.StatusServiceUnavailable})
		return
	}
	lat, lon, err := validator.ParseAndValidateCoordinates(c.Query("lat"), c.Query("lon"))
	if err != nil {
		badWeatherRequest(c, err.Error())
		return
	}
	weatherType := c.DefaultQuery("type", "current")
	if weatherType != "current" && weatherType != "forecast" {
		badWeatherRequest(c, `typeは "current" または "forecast" である必要があります`)
		return
	}
	units := c.DefaultQuery("units", "metric")
	if !slices.Contains(weatherUnits, units) {
		badWeatherRequest(c, `unitsは "standard", "metric", または "imperial" である必要があります`)
		return
	}

	ctx := c.Request.Context()
	var (
		data      any
		fromCache bool
	)
	if weatherType == "forecast" {
		data, fromCache, err = h.usecase.GetForecast(ctx, lat, lon, units)
	} else {
		var w *entity.WeatherData
		w, fromCache, err = h.usecase.GetCurrent(ctx, lat, lon, units)
		if err == nil {
			data = toWeatherBody(w)
		}
	}
	if err != nil {
		if respondClassifiedError(c, err, gin.H{"success": false}) {
			return
		}
		slog.ErrorContext(ctx, "Weather API unexpected error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "天気データの取得中にエラーが発生しました", "code": "INTERNAL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, weatherResponse{
		Success: true, Type: weatherType, Data: data, FromCache: fromCache, FetchedAt: isoTime(time.Now()),
	})
}

func toWeatherBody(w *entity.WeatherData) weatherBody {
	b := weatherBody{
		CityName: w.CityName, Country: w.Country,
		Temperature: w.Temperature, FeelsLike: w.FeelsLike, TempMin: w.TempMin, TempMax: w.TempMax,
		Humidity: w.Humidity, Pressure: w.Pressure,
		WindSpeed: w.WindSpeed, WindDeg: w.WindDeg, WindGust: w.WindGust,
		Cloudiness: w.Cloudiness, Visibility: w.Visibility,
		Rain: w.Rain, Snow: w.Snow,
		Sunrise: isoTime(w.Sunrise), Sunset: isoTime(w.Sunset), DataTime: isoTime(w.DateTime),
		Timezone: w.Timezone,
	}
	b.Coordinates.Lat, b.Coordinates.Lon = w.Lat, w.Lon
	b.Weather.ID, b.Weather.Main = w.Condition.ID, w.Condition.Main
	b.Weather.Description, b.Weather.Icon = w.Condition.Description, w.Condition.Icon
	return b
}
