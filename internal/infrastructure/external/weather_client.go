package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

// WeatherClient は OpenWeatherMap（2.5）のクライアント。
type WeatherClient struct {
	apiKey  string
	baseURL string
	fetch   *fetcher
}

func NewWeatherClient(apiKey string) *WeatherClient {
	return newWeatherClientWithHTTPClient(apiKey, "https://api.openweathermap.org", &http.Client{})
}

func newWeatherClientWithHTTPClient(apiKey, baseURL string, httpClient *http.Client) *WeatherClient {
	return &WeatherClient{apiKey: apiKey, baseURL: baseURL, fetch: newFetcher(httpClient)}
}

// owmCurrentResponse は /data/2.5/weather の応答。形は testdata/owm_current.json を正とする。
// 風速と気圧はスコアに使うので、欠け・null を見分けられるようポインタで受ける。
type owmCurrentResponse struct {
	Coord struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"coord"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Main *struct {
		Temp      float64  `json:"temp"`
		FeelsLike float64  `json:"feels_like"`
		TempMin   *float64 `json:"temp_min"`
		TempMax   *float64 `json:"temp_max"`
		Pressure  *float64 `json:"pressure"`
		Humidity  int      `json:"humidity"`
	} `json:"main"`
	Visibility int `json:"visibility"`
	Wind       *struct {
		Speed *float64 `json:"speed"`
		Deg   int      `json:"deg"`
		Gust  *float64 `json:"gust"`
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Rain *struct {
		OneHour *float64 `json:"1h"`
	} `json:"rain"`
	Snow *struct {
		OneHour *float64 `json:"1h"`
	} `json:"snow"`
	Dt  int64 `json:"dt"`
	Sys struct {
		Country string `json:"country"`
		Sunrise int64  `json:"sunrise"`
		Sunset  int64  `json:"sunset"`
	} `json:"sys"`
	Timezone int    `json:"timezone"`
	Name     string `json:"name"`
}

func (c *WeatherClient) url(path string, lat, lon float64, units string) string {
	q := url.Values{
		"lat":   {strconv.FormatFloat(lat, 'f', -1, 64)},
		"lon":   {strconv.FormatFloat(lon, 'f', -1, 64)},
		"units": {units},
		"lang":  {"ja"},
		"appid": {c.apiKey},
	}
	// 注意: URL には API キー（appid）が入るので、ログにもエラーにも含めない（getJSON が外す）。
	return c.baseURL + path + "?" + q.Encode()
}

// FetchCurrent は現在の天気を取得する。風速・気圧が数でなければ 502 の上流の失敗にする（Next #414）。
func (c *WeatherClient) FetchCurrent(ctx context.Context, lat, lon float64, units string) (*entity.WeatherData, error) {
	var raw owmCurrentResponse
	if err := c.fetch.getJSON(ctx, c.url("/data/2.5/weather", lat, lon, units), &raw); err != nil {
		return nil, fmt.Errorf("openweathermap current: %w", err)
	}
	switch {
	case raw.Wind == nil || raw.Wind.Speed == nil:
		return nil, domain.NewInvalidUpstreamResponse("OpenWeatherMap returned an invalid wind.speed")
	case raw.Main == nil || raw.Main.Pressure == nil:
		return nil, domain.NewInvalidUpstreamResponse("OpenWeatherMap returned an invalid main.pressure")
	case len(raw.Weather) == 0:
		return nil, domain.NewInvalidUpstreamResponse("OpenWeatherMap returned no weather condition")
	}

	w := &entity.WeatherData{
		Lat:         raw.Coord.Lat,
		Lon:         raw.Coord.Lon,
		CityName:    raw.Name,
		Country:     raw.Sys.Country,
		Temperature: raw.Main.Temp,
		FeelsLike:   raw.Main.FeelsLike,
		TempMin:     orDefault(raw.Main.TempMin, raw.Main.Temp),
		TempMax:     orDefault(raw.Main.TempMax, raw.Main.Temp),
		Humidity:    raw.Main.Humidity,
		Pressure:    *raw.Main.Pressure,
		WindSpeed:   *raw.Wind.Speed,
		WindDeg:     raw.Wind.Deg,
		WindGust:    raw.Wind.Gust,
		Cloudiness:  raw.Clouds.All,
		Visibility:  raw.Visibility,
		Condition: entity.WeatherCondition{
			ID:          raw.Weather[0].ID,
			Main:        raw.Weather[0].Main,
			Description: raw.Weather[0].Description,
			Icon:        raw.Weather[0].Icon,
		},
		Sunrise:  time.Unix(raw.Sys.Sunrise, 0),
		Sunset:   time.Unix(raw.Sys.Sunset, 0),
		DateTime: time.Unix(raw.Dt, 0),
		Timezone: raw.Timezone,
	}
	if raw.Rain != nil {
		w.Rain = raw.Rain.OneHour
	}
	if raw.Snow != nil {
		w.Snow = raw.Snow.OneHour
	}
	return w, nil
}

// FetchForecast は5日間・3時間ごとの予報を、上流の応答のまま返す（Next も整形せずに返している）。
func (c *WeatherClient) FetchForecast(ctx context.Context, lat, lon float64, units string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.fetch.getJSON(ctx, c.url("/data/2.5/forecast", lat, lon, units), &raw); err != nil {
		return nil, fmt.Errorf("openweathermap forecast: %w", err)
	}
	return raw, nil
}

func orDefault(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}
