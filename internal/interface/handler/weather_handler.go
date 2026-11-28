package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/openapi"
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

func weatherBadRequest(msg string) openapi.GetWeather400JSONResponse {
	code := openapi.INVALIDPARAMS
	status := http.StatusBadRequest
	return openapi.GetWeather400JSONResponse{
		WeatherErrorJSONResponse: openapi.WeatherErrorJSONResponse{
			Error:  msg,
			Code:   &code,
			Status: &status,
		},
	}
}

func weatherNotConfigured(ctx context.Context) openapi.GetWeather503JSONResponse {
	slog.ErrorContext(ctx, "OPENWEATHER_API_KEY is not configured")
	code := openapi.APINOTCONFIGURED
	status := http.StatusServiceUnavailable
	return openapi.GetWeather503JSONResponse{
		Error:  "Weather API is not configured",
		Code:   &code,
		Status: &status,
	}
}

func (h *WeatherHandler) GetWeather(ctx context.Context, req openapi.GetWeatherRequestObject) (openapi.GetWeatherResponseObject, error) {
	reqCtx, path := requestContext(ctx)

	// Next と同じく、クエリの検証とキャッシュより前に確かめる。
	if !h.apiConfigured {
		return weatherNotConfigured(reqCtx), nil
	}
	if err := validator.ValidateCoordinates(req.Params.Lat, req.Params.Lon); err != nil {
		return weatherBadRequest(err.Error()), nil
	}

	weatherType := string(openapi.GetWeatherParamsTypeCurrent)
	if req.Params.Type != nil && *req.Params.Type != "" {
		weatherType = string(*req.Params.Type)
	}
	if weatherType != "current" && weatherType != "forecast" {
		return weatherBadRequest(`typeは "current" または "forecast" である必要があります`), nil
	}

	units := string(openapi.Metric)
	if req.Params.Units != nil && *req.Params.Units != "" {
		units = string(*req.Params.Units)
	}
	if !slices.Contains(weatherUnits, units) {
		return weatherBadRequest(`unitsは "standard", "metric", または "imperial" である必要があります`), nil
	}

	if weatherType == "forecast" {
		raw, fromCache, err := h.usecase.GetForecast(reqCtx, req.Params.Lat, req.Params.Lon, units)
		if err != nil {
			return h.handleWeatherError(reqCtx, path, err), nil
		}
		var resp openapi.WeatherResponse
		if err := resp.FromWeatherForecastResponse(openapi.WeatherForecastResponse{
			Success:   true,
			Type:      openapi.WeatherForecastResponseTypeForecast,
			Data:      raw,
			FromCache: fromCache,
			FetchedAt: isoTime(time.Now()),
		}); err != nil {
			return nil, fmt.Errorf("marshaling weather forecast response: %w", err)
		}
		return openapi.GetWeather200JSONResponse(resp), nil
	}

	w, fromCache, err := h.usecase.GetCurrent(reqCtx, req.Params.Lat, req.Params.Lon, units)
	if err != nil {
		return h.handleWeatherError(reqCtx, path, err), nil
	}
	var resp openapi.WeatherResponse
	if err := resp.FromWeatherCurrentResponse(openapi.WeatherCurrentResponse{
		Success:   true,
		Type:      openapi.WeatherCurrentResponseTypeCurrent,
		Data:      toWeatherCurrent(w),
		FromCache: fromCache,
		FetchedAt: isoTime(time.Now()),
	}); err != nil {
		return nil, fmt.Errorf("marshaling weather current response: %w", err)
	}
	return openapi.GetWeather200JSONResponse(resp), nil
}

func (h *WeatherHandler) handleWeatherError(ctx context.Context, path string, err error) openapi.GetWeatherResponseObject {
	if status, resp, ok := classifyError(ctx, path, err); ok {
		return weatherStatusFromClassified(status, resp)
	}
	slog.ErrorContext(ctx, "Weather API unexpected error", "error", err)
	f := false
	code := openapi.INTERNALERROR
	return openapi.GetWeather500JSONResponse{
		Success: &f,
		Error:   "天気データの取得中にエラーが発生しました",
		Code:    &code,
	}
}

func toWeatherCurrent(w *entity.WeatherData) openapi.WeatherCurrent {
	return openapi.WeatherCurrent{
		Coordinates: openapi.Coordinates{
			Lat: w.Lat,
			Lon: w.Lon,
		},
		CityName:    w.CityName,
		Country:     w.Country,
		Temperature: w.Temperature,
		FeelsLike:   w.FeelsLike,
		TempMin:     w.TempMin,
		TempMax:     w.TempMax,
		Humidity:    w.Humidity,
		Pressure:    w.Pressure,
		WindSpeed:   w.WindSpeed,
		WindDeg:     w.WindDeg,
		WindGust:    w.WindGust,
		Cloudiness:  w.Cloudiness,
		Visibility:  w.Visibility,
		Weather: openapi.WeatherCondition{
			Id:          w.Condition.ID,
			Main:        w.Condition.Main,
			Description: w.Condition.Description,
			Icon:        w.Condition.Icon,
		},
		Rain:     w.Rain,
		Snow:     w.Snow,
		Sunrise:  isoTime(w.Sunrise),
		Sunset:   isoTime(w.Sunset),
		DataTime: isoTime(w.DateTime),
		Timezone: w.Timezone,
	}
}
