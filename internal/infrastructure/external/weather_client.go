package external

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type WeatherClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewWeatherClient(apiKey string) *WeatherClient {
	return &WeatherClient{
		apiKey:  apiKey,
		baseURL: "https://api.openweathermap.org",
		client:  newHTTPClient(),
	}
}

func newWeatherClientWithHTTPClient(apiKey, baseURL string, httpClient *http.Client) *WeatherClient {
	return &WeatherClient{apiKey: apiKey, baseURL: baseURL, client: httpClient}
}

type owmCurrentResponse struct {
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Pressure  float64 `json:"pressure"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
	Dt int64 `json:"dt"`
}

func (c *WeatherClient) FetchCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error) {
	url := fmt.Sprintf("%s/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric&lang=ja",
		c.baseURL, lat, lon, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating weather request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather API request failed: %w", err)
	}
	// 注意: ログには URL を含めない。OpenWeather の URL クエリには
	// API キー (`appid=...`) が含まれるため、ログ流出すると認証情報漏洩リスクがある。
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Warn("failed to close weather response body", "error", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	var data owmCurrentResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode weather response: %w", err)
	}

	description := ""
	if len(data.Weather) > 0 {
		description = data.Weather[0].Description
	}

	return &entity.WeatherData{
		Temperature: data.Main.Temp,
		FeelsLike:   data.Main.FeelsLike,
		WindSpeed:   data.Wind.Speed,
		WindDeg:     data.Wind.Deg,
		Pressure:    data.Main.Pressure,
		Humidity:    data.Main.Humidity,
		Description: description,
		DateTime:    time.Unix(data.Dt, 0),
	}, nil
}

type owmForecastResponse struct {
	List []struct {
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Pressure  float64 `json:"pressure"`
			Humidity  int     `json:"humidity"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		} `json:"wind"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Dt int64 `json:"dt"`
	} `json:"list"`
}

func (c *WeatherClient) FetchForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error) {
	url := fmt.Sprintf("%s/data/2.5/forecast?lat=%f&lon=%f&appid=%s&units=metric&lang=ja",
		c.baseURL, lat, lon, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating forecast request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forecast API request failed: %w", err)
	}
	// 注意: ログには URL を含めない。OpenWeather の URL クエリには
	// API キー (`appid=...`) が含まれるため、ログ流出すると認証情報漏洩リスクがある。
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Warn("failed to close forecast response body", "error", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast API returned status %d", resp.StatusCode)
	}

	var data owmForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode forecast response: %w", err)
	}

	result := make([]*entity.WeatherData, 0, len(data.List))
	for _, item := range data.List {
		description := ""
		if len(item.Weather) > 0 {
			description = item.Weather[0].Description
		}
		result = append(result, &entity.WeatherData{
			Temperature: item.Main.Temp,
			FeelsLike:   item.Main.FeelsLike,
			WindSpeed:   item.Wind.Speed,
			WindDeg:     item.Wind.Deg,
			Pressure:    item.Main.Pressure,
			Humidity:    item.Main.Humidity,
			Description: description,
			DateTime:    time.Unix(item.Dt, 0),
		})
	}
	return result, nil
}
