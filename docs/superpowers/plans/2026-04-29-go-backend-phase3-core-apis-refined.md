# Go Backend Phase 3: Core APIs Implementation Plan (Refined)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 天気・潮汐・場所検索・釣りやすさスコア・お気に入りCRUDのAPIを実装し、FishingConditionsAppフロントエンドと同じエンドポイントで動作させる。

**Architecture:** `infrastructure/external`（外部APIクライアント）→ `usecase`（Redisキャッシュ付きビジネスロジック）→ `interface/handler`（HTTPハンドラー）の流れ。スコア算出は外部APIを呼ばず天気・潮汐データから純粋計算。

**Tech Stack:** Go 1.26, Gin, go-redis/v9, encoding/json, net/http

**前提条件:** Phase 1・2完了済み（認証・DB・Redis・ルーター動作確認済み）

---

## 元のプランからの差分（重要）

1. **`*sql.DB` 使用**: Phase 2 で `pgxpool.Pool` から `*sql.DB` に変更済み。`favorite_repository.go` も `*sql.DB` を使う。
2. **`LocationSearchResult` は domain/entity に定義**: `usecase/location` と `interface/handler` から `infrastructure/external` を import するのはクリーンアーキテクチャ違反。`entity.LocationSearchResult` として定義し、全レイヤーから domain 経由で参照する。
3. **`toLocationEntity` の nullable 対応**: `sqlcgen.Location` の `Region`/`Prefecture` は `sql.NullString`、`PortID` は `uuid.NullUUID`。null チェックが必要。
4. **`router.New` に `*jwtutil.Manager` を渡す**: protected routes で JWTAuth ミドルウェアを使うために必要。
5. **`main.go` は slog を維持**: Phase 2 の refinement 通り `slog.Error` + `os.Exit(1)` を維持。`jwtManagerAdapter` も継続使用。
6. **`domain/repository.Cache` インターフェース活用**: 既に `domain/repository/cache.go` に `Cache` インターフェースが定義済み。usecase の mock にはローカル `Cache` インターフェース（Get/Set のみ）を使い、テストを最小限に保つ。
7. **`errors.Is` 使用**: sentinel error 比較は必ず `errors.Is` を使う（`==` 禁止）。

---

## ファイル構成

| 操作 | ファイル | 内容 |
|------|---------|------|
| 変更 | `internal/domain/entity/location.go` | `LocationSearchResult` 追加 |
| 新規作成 | `internal/infrastructure/external/retry.go` | リトライ付きHTTPトランスポート |
| 新規作成 | `internal/infrastructure/external/weather_client.go` | OpenWeatherMap APIクライアント |
| 新規作成 | `internal/infrastructure/external/weather_client_test.go` | 天気クライアントテスト |
| 新規作成 | `internal/infrastructure/external/tide_client.go` | tide736.net APIクライアント |
| 新規作成 | `internal/infrastructure/external/tide_client_test.go` | 潮汐クライアントテスト |
| 新規作成 | `internal/infrastructure/external/maps_client.go` | Google Maps Geocoding APIクライアント |
| 新規作成 | `internal/usecase/weather/weather.go` | 天気ユースケース（Redisキャッシュ付き） |
| 新規作成 | `internal/usecase/weather/weather_test.go` | 天気ユースケーステスト |
| 新規作成 | `internal/usecase/tide/tide.go` | 潮汐ユースケース（Redisキャッシュ付き） |
| 新規作成 | `internal/usecase/tide/tide_test.go` | 潮汐ユースケーステスト |
| 新規作成 | `internal/usecase/location/location.go` | 場所検索ユースケース |
| 新規作成 | `internal/usecase/location/location_test.go` | 場所検索ユースケーステスト |
| 新規作成 | `internal/usecase/score/score.go` | 釣りやすさスコア算出ユースケース |
| 新規作成 | `internal/usecase/score/score_test.go` | スコア算出テスト |
| 新規作成 | `internal/infrastructure/db/favorite_repository.go` | FavoriteRepository sqlc実装（`*sql.DB` 使用） |
| 新規作成 | `internal/usecase/favorite/favorite.go` | お気に入りユースケース |
| 新規作成 | `internal/usecase/favorite/favorite_test.go` | お気に入りユースケーステスト |
| 新規作成 | `internal/interface/handler/weather_handler.go` | 天気ハンドラー |
| 新規作成 | `internal/interface/handler/tide_handler.go` | 潮汐ハンドラー |
| 新規作成 | `internal/interface/handler/location_handler.go` | 場所検索ハンドラー |
| 新規作成 | `internal/interface/handler/favorite_handler.go` | お気に入りハンドラー |
| 変更 | `internal/interface/router/router.go` | 全APIルート追加・`*jwtutil.Manager` 受け取り |
| 変更 | `cmd/server/main.go` | 全DI追加・slog 維持 |

---

## Task 0: `LocationSearchResult` をドメインエンティティに追加

**Files:**
- Modify: `internal/domain/entity/location.go`

場所検索結果（Google Maps Geocoding API の返り値）は DB の `Location` エンティティとは別物。
`usecase/location` と `interface/handler` が `infrastructure/external` を import するのはクリーンアーキテクチャ違反なので、domain/entity に定義する。

- [ ] **Step 1: `LocationSearchResult` を `location.go` に追加**

`internal/domain/entity/location.go` の末尾に追加する。

```go
type LocationSearchResult struct {
	Name       string
	Latitude   float64
	Longitude  float64
	Prefecture string
	Region     string
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/domain/...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/domain/entity/location.go
git commit -m "feat: LocationSearchResult をドメインエンティティに追加"
```

---

## Task 1: リトライ付きHTTPクライアント

**Files:**
- Create: `internal/infrastructure/external/retry.go`

- [ ] **Step 1: 実装**

```go
// internal/infrastructure/external/retry.go
package external

import (
	"net/http"
	"time"
)

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
}

func newRetryTransport(maxRetries int) http.RoundTripper {
	return &retryTransport{
		base:       http.DefaultTransport,
		maxRetries: maxRetries,
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	for i := 0; i <= t.maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * 500 * time.Millisecond)
		}
		resp, err := t.base.RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = nil
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	return nil, lastErr
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: newRetryTransport(3),
	}
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/infrastructure/external/...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/infrastructure/external/retry.go
git commit -m "feat: リトライ付きHTTPクライアント追加"
```

---

## Task 2: OpenWeatherMap APIクライアント

**Files:**
- Create: `internal/infrastructure/external/weather_client.go`
- Create: `internal/infrastructure/external/weather_client_test.go`

- [ ] **Step 1: テストを書く**

```go
// internal/infrastructure/external/weather_client_test.go
package external

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeatherClient_FetchCurrent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/data/2.5/weather", r.URL.Path)
		assert.Equal(t, "35.689500", r.URL.Query().Get("lat"))
		assert.Equal(t, "139.691700", r.URL.Query().Get("lon"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"main": map[string]interface{}{
				"temp":       20.5,
				"feels_like": 19.0,
				"pressure":   1013.0,
				"humidity":   65,
			},
			"wind": map[string]interface{}{
				"speed": 3.5,
				"deg":   180,
			},
			"weather": []map[string]interface{}{
				{"description": "晴れ"},
			},
			"dt": 1700000000,
		})
	}))
	defer server.Close()

	client := newWeatherClientWithBaseURL("test-api-key", server.URL)
	result, err := client.FetchCurrent(35.6895, 139.6917)
	require.NoError(t, err)
	assert.Equal(t, 20.5, result.Temperature)
	assert.Equal(t, 3.5, result.WindSpeed)
	assert.Equal(t, "晴れ", result.Description)
}

func TestWeatherClient_FetchForecast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/data/2.5/forecast", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"list": []map[string]interface{}{
				{
					"main":    map[string]interface{}{"temp": 18.0, "feels_like": 17.0, "pressure": 1010.0, "humidity": 70},
					"wind":    map[string]interface{}{"speed": 2.0, "deg": 90},
					"weather": []map[string]interface{}{{"description": "曇り"}},
					"dt":      1700003600,
				},
			},
		})
	}))
	defer server.Close()

	client := newWeatherClientWithBaseURL("test-api-key", server.URL)
	results, err := client.FetchForecast(35.6895, 139.6917)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 18.0, results[0].Temperature)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/infrastructure/external/... -v -run TestWeatherClient
```

Expected: FAIL（パッケージが存在しない）

- [ ] **Step 3: 実装**

```go
// internal/infrastructure/external/weather_client.go
package external

import (
	"encoding/json"
	"fmt"
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

func newWeatherClientWithBaseURL(apiKey, baseURL string) *WeatherClient {
	return &WeatherClient{apiKey: apiKey, baseURL: baseURL, client: newHTTPClient()}
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

func (c *WeatherClient) FetchCurrent(lat, lon float64) (*entity.WeatherData, error) {
	url := fmt.Sprintf("%s/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric&lang=ja",
		c.baseURL, lat, lon, c.apiKey)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("weather API request failed: %w", err)
	}
	defer resp.Body.Close()

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

func (c *WeatherClient) FetchForecast(lat, lon float64) ([]*entity.WeatherData, error) {
	url := fmt.Sprintf("%s/data/2.5/forecast?lat=%f&lon=%f&appid=%s&units=metric&lang=ja",
		c.baseURL, lat, lon, c.apiKey)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("forecast API request failed: %w", err)
	}
	defer resp.Body.Close()

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
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/infrastructure/external/... -v -run TestWeatherClient
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/infrastructure/external/weather_client.go internal/infrastructure/external/weather_client_test.go
git commit -m "feat: OpenWeatherMap APIクライアント追加"
```

---

## Task 3: tide736.net APIクライアント

**Files:**
- Create: `internal/infrastructure/external/tide_client.go`
- Create: `internal/infrastructure/external/tide_client_test.go`

- [ ] **Step 1: テストを書く**

```go
// internal/infrastructure/external/tide_client_test.go
package external

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTideClient_FetchTideData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tide_type": "大潮",
			"high_tides": []map[string]interface{}{
				{"time": "06:30", "height": 185.0},
				{"time": "18:45", "height": 192.0},
			},
			"low_tides": []map[string]interface{}{
				{"time": "12:15", "height": 22.0},
			},
		})
	}))
	defer server.Close()

	client := newTideClientWithBaseURL(server.URL)
	result, err := client.FetchTideData("13", "TK", "2026-04-29")
	require.NoError(t, err)
	assert.Equal(t, "大潮", result.TideType)
	assert.Len(t, result.HighTides, 2)
	assert.Len(t, result.LowTides, 1)
	assert.Equal(t, 185.0, result.HighTides[0].Height)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/infrastructure/external/... -v -run TestTideClient
```

Expected: FAIL

- [ ] **Step 3: 実装**

```go
// internal/infrastructure/external/tide_client.go
package external

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type TideClient struct {
	baseURL string
	client  *http.Client
}

func NewTideClient() *TideClient {
	return &TideClient{
		baseURL: "https://tide736.net",
		client:  newHTTPClient(),
	}
}

func newTideClientWithBaseURL(baseURL string) *TideClient {
	return &TideClient{baseURL: baseURL, client: newHTTPClient()}
}

type tideAPIResponse struct {
	TideType  string `json:"tide_type"`
	HighTides []struct {
		Time   string  `json:"time"`
		Height float64 `json:"height"`
	} `json:"high_tides"`
	LowTides []struct {
		Time   string  `json:"time"`
		Height float64 `json:"height"`
	} `json:"low_tides"`
}

func (c *TideClient) FetchTideData(prefCode, portCode, date string) (*entity.TideData, error) {
	url := fmt.Sprintf("%s/api/get_tide/%s/%s/%s/",
		c.baseURL, prefCode, portCode, date)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("tide API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tide API returned status %d", resp.StatusCode)
	}

	var data tideAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode tide response: %w", err)
	}

	tideData := &entity.TideData{
		PortCode: portCode,
		Date:     date,
		TideType: data.TideType,
	}

	loc, _ := time.LoadLocation("Asia/Tokyo")
	dateBase, _ := time.ParseInLocation("2006-01-02", date, loc)

	for _, h := range data.HighTides {
		t, err := parseTimeOnDate(dateBase, h.Time, loc)
		if err != nil {
			continue
		}
		tideData.HighTides = append(tideData.HighTides, entity.TideEvent{Time: t, Height: h.Height})
	}
	for _, l := range data.LowTides {
		t, err := parseTimeOnDate(dateBase, l.Time, loc)
		if err != nil {
			continue
		}
		tideData.LowTides = append(tideData.LowTides, entity.TideEvent{Time: t, Height: l.Height})
	}

	return tideData, nil
}

func parseTimeOnDate(base time.Time, timeStr string, loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation("15:04", timeStr, loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(base.Year(), base.Month(), base.Day(), t.Hour(), t.Minute(), 0, 0, loc), nil
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/infrastructure/external/... -v -run TestTideClient
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/infrastructure/external/tide_client.go internal/infrastructure/external/tide_client_test.go
git commit -m "feat: tide736.net APIクライアント追加"
```

---

## Task 4: Google Maps APIクライアント

**Files:**
- Create: `internal/infrastructure/external/maps_client.go`

**注意:** `LocationResult` は定義しない。Task 0 で追加した `entity.LocationSearchResult` を使う。

- [ ] **Step 1: 実装**

```go
// internal/infrastructure/external/maps_client.go
package external

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type MapsClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewMapsClient(apiKey string) *MapsClient {
	return &MapsClient{
		apiKey:  apiKey,
		baseURL: "https://maps.googleapis.com",
		client:  newHTTPClient(),
	}
}

type geocodeResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		AddressComponents []struct {
			LongName string   `json:"long_name"`
			Types    []string `json:"types"`
		} `json:"address_components"`
	} `json:"results"`
	Status string `json:"status"`
}

func (c *MapsClient) SearchLocations(query string, limit int) ([]*entity.LocationSearchResult, error) {
	params := url.Values{}
	params.Set("address", query)
	params.Set("key", c.apiKey)
	params.Set("language", "ja")
	params.Set("region", "jp")

	apiURL := fmt.Sprintf("%s/maps/api/geocode/json?%s", c.baseURL, params.Encode())

	resp, err := c.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("maps API request failed: %w", err)
	}
	defer resp.Body.Close()

	var data geocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode maps response: %w", err)
	}

	if data.Status != "OK" && data.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("maps API error: %s", data.Status)
	}

	results := make([]*entity.LocationSearchResult, 0)
	for i, r := range data.Results {
		if i >= limit {
			break
		}
		result := &entity.LocationSearchResult{
			Name:      r.FormattedAddress,
			Latitude:  r.Geometry.Location.Lat,
			Longitude: r.Geometry.Location.Lng,
		}
		for _, comp := range r.AddressComponents {
			for _, t := range comp.Types {
				if t == "administrative_area_level_1" {
					result.Prefecture = comp.LongName
				}
			}
		}
		results = append(results, result)
	}
	return results, nil
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/infrastructure/external/...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/infrastructure/external/maps_client.go
git commit -m "feat: Google Maps APIクライアント追加"
```

---

## Task 5: 天気ユースケース（Redisキャッシュ付き）

**Files:**
- Create: `internal/usecase/weather/weather.go`
- Create: `internal/usecase/weather/weather_test.go`

- [ ] **Step 1: テストを書く**

```go
// internal/usecase/weather/weather_test.go
package weather

import (
	"context"
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockWeatherAPI struct{ mock.Mock }

func (m *MockWeatherAPI) FetchCurrent(lat, lon float64) (*entity.WeatherData, error) {
	args := m.Called(lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.WeatherData), args.Error(1)
}
func (m *MockWeatherAPI) FetchForecast(lat, lon float64) ([]*entity.WeatherData, error) {
	args := m.Called(lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.WeatherData), args.Error(1)
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
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func TestWeatherUsecase_GetCurrent_CacheMiss(t *testing.T) {
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	weatherData := &entity.WeatherData{Temperature: 22.0, WindSpeed: 3.5, Description: "晴れ"}
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, 30*time.Minute).Return(nil)
	mockAPI.On("FetchCurrent", 35.6895, 139.6917).Return(weatherData, nil)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetCurrent(context.Background(), 35.6895, 139.6917)

	require.NoError(t, err)
	assert.Equal(t, 22.0, result.Temperature)
	mockAPI.AssertCalled(t, "FetchCurrent", 35.6895, 139.6917)
	mockCache.AssertExpectations(t)
}

func TestWeatherUsecase_GetCurrent_CacheHit(t *testing.T) {
	mockAPI := &MockWeatherAPI{}
	mockCache := &MockCache{}

	cachedJSON := []byte(`{"Temperature":20.0,"WindSpeed":2.0,"Description":"曇り","FeelsLike":19.0,"WindDeg":90,"Pressure":1010,"Humidity":70,"DateTime":"2026-04-29T10:00:00Z"}`)
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(cachedJSON, nil)

	uc := NewWeatherUsecase(mockAPI, mockCache)
	result, err := uc.GetCurrent(context.Background(), 35.6895, 139.6917)

	require.NoError(t, err)
	assert.Equal(t, 20.0, result.Temperature)
	mockAPI.AssertNotCalled(t, "FetchCurrent")
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/usecase/weather/... -v
```

Expected: FAIL

- [ ] **Step 3: 実装**

```go
// internal/usecase/weather/weather.go
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/pkg/validator"
)

type WeatherAPI interface {
	FetchCurrent(lat, lon float64) (*entity.WeatherData, error)
	FetchForecast(lat, lon float64) ([]*entity.WeatherData, error)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

const weatherTTL = 30 * time.Minute

type WeatherUsecase struct {
	api   WeatherAPI
	cache Cache
}

func NewWeatherUsecase(api WeatherAPI, cache Cache) *WeatherUsecase {
	return &WeatherUsecase{api: api, cache: cache}
}

func (u *WeatherUsecase) GetCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error) {
	key := cacheKey(lat, lon, "current")

	if cached, _ := u.cache.Get(ctx, key); cached != nil {
		var data entity.WeatherData
		if err := json.Unmarshal(cached, &data); err == nil {
			return &data, nil
		}
	}

	data, err := u.api.FetchCurrent(lat, lon)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, weatherTTL)
	}

	return data, nil
}

func (u *WeatherUsecase) GetForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error) {
	key := cacheKey(lat, lon, "forecast")

	if cached, _ := u.cache.Get(ctx, key); cached != nil {
		var data []*entity.WeatherData
		if err := json.Unmarshal(cached, &data); err == nil {
			return data, nil
		}
	}

	data, err := u.api.FetchForecast(lat, lon)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, weatherTTL)
	}

	return data, nil
}

func cacheKey(lat, lon float64, typ string) string {
	roundedLat := validator.RoundCoordinate(lat, 4)
	roundedLon := validator.RoundCoordinate(lon, 4)
	return fmt.Sprintf("weather:%s:%.4f:%.4f", typ, roundedLat, roundedLon)
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/usecase/weather/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/usecase/weather/
git commit -m "feat: 天気ユースケース実装（Redisキャッシュ付き）"
```

---

## Task 6: 潮汐ユースケース（Redisキャッシュ付き）

**Files:**
- Create: `internal/usecase/tide/tide.go`
- Create: `internal/usecase/tide/tide_test.go`

- [ ] **Step 1: テストを書く**

```go
// internal/usecase/tide/tide_test.go
package tide

import (
	"context"
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockTideAPI struct{ mock.Mock }

func (m *MockTideAPI) FetchTideData(prefCode, portCode, date string) (*entity.TideData, error) {
	args := m.Called(prefCode, portCode, date)
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
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func TestTideUsecase_GetTideData_CacheMiss(t *testing.T) {
	mockAPI := &MockTideAPI{}
	mockCache := &MockCache{}

	tideData := &entity.TideData{
		PortCode: "TK",
		Date:     "2026-04-29",
		TideType: "大潮",
		HighTides: []entity.TideEvent{{Height: 185.0}},
	}

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, 6*time.Hour).Return(nil)
	mockAPI.On("FetchTideData", "13", "TK", "2026-04-29").Return(tideData, nil)

	uc := NewTideUsecase(mockAPI, mockCache)
	result, err := uc.GetTideData(context.Background(), "13", "TK", "2026-04-29")

	require.NoError(t, err)
	assert.Equal(t, "大潮", result.TideType)
	mockAPI.AssertCalled(t, "FetchTideData", "13", "TK", "2026-04-29")
	mockCache.AssertExpectations(t)
}

func TestTideUsecase_GetTideData_CacheHit(t *testing.T) {
	mockAPI := &MockTideAPI{}
	mockCache := &MockCache{}

	cachedJSON := []byte(`{"PortCode":"TK","Date":"2026-04-29","HighTides":[],"LowTides":[],"TideType":"中潮"}`)
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(cachedJSON, nil)

	uc := NewTideUsecase(mockAPI, mockCache)
	result, err := uc.GetTideData(context.Background(), "13", "TK", "2026-04-29")

	require.NoError(t, err)
	assert.Equal(t, "中潮", result.TideType)
	mockAPI.AssertNotCalled(t, "FetchTideData")
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/usecase/tide/... -v
```

Expected: FAIL

- [ ] **Step 3: 実装**

```go
// internal/usecase/tide/tide.go
package tide

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type TideAPI interface {
	FetchTideData(prefCode, portCode, date string) (*entity.TideData, error)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

const tideTTL = 6 * time.Hour

type TideUsecase struct {
	api   TideAPI
	cache Cache
}

func NewTideUsecase(api TideAPI, cache Cache) *TideUsecase {
	return &TideUsecase{api: api, cache: cache}
}

func (u *TideUsecase) GetTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error) {
	key := fmt.Sprintf("tide:%s:%s", portCode, date)

	if cached, _ := u.cache.Get(ctx, key); cached != nil {
		var data entity.TideData
		if err := json.Unmarshal(cached, &data); err == nil {
			return &data, nil
		}
	}

	data, err := u.api.FetchTideData(prefCode, portCode, date)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(data); err == nil {
		u.cache.Set(ctx, key, b, tideTTL)
	}

	return data, nil
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/usecase/tide/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/usecase/tide/
git commit -m "feat: 潮汐ユースケース実装（Redisキャッシュ付き）"
```

---

## Task 7: 釣りやすさスコア算出ユースケース

**Files:**
- Create: `internal/usecase/score/score.go`
- Create: `internal/usecase/score/score_test.go`

- [ ] **Step 1: テストを書く**

```go
// internal/usecase/score/score_test.go
package score

import (
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestCalculate_GoodConditions(t *testing.T) {
	uc := NewScoreUsecase()

	// 早朝・大潮・穏やかな天気
	morningTime := time.Date(2026, 4, 29, 6, 0, 0, 0, time.UTC)
	weather := &entity.WeatherData{
		WindSpeed:   2.0,
		Description: "晴れ",
		Pressure:    1015.0,
	}
	tide := &entity.TideData{
		TideType: "大潮",
		HighTides: []entity.TideEvent{
			{Time: morningTime.Add(-1 * time.Hour), Height: 185.0},
		},
	}

	result := uc.Calculate(weather, tide, morningTime)
	assert.Greater(t, result.Total, 60)
	assert.Equal(t, entity.ScoreRankGood, result.Rank)
}

func TestCalculate_PoorConditions(t *testing.T) {
	uc := NewScoreUsecase()

	// 日中・小潮・強風
	middayTime := time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)
	weather := &entity.WeatherData{
		WindSpeed:   12.0,
		Description: "暴風雨",
		Pressure:    985.0,
	}
	tide := &entity.TideData{
		TideType:  "小潮",
		HighTides: []entity.TideEvent{},
		LowTides:  []entity.TideEvent{},
	}

	result := uc.Calculate(weather, tide, middayTime)
	assert.Less(t, result.Total, 40)
}

func TestGetScoreRank(t *testing.T) {
	assert.Equal(t, entity.ScoreRankExcellent, entity.GetScoreRank(85))
	assert.Equal(t, entity.ScoreRankGood, entity.GetScoreRank(65))
	assert.Equal(t, entity.ScoreRankFair, entity.GetScoreRank(45))
	assert.Equal(t, entity.ScoreRankPoor, entity.GetScoreRank(25))
	assert.Equal(t, entity.ScoreRankBad, entity.GetScoreRank(10))
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/usecase/score/... -v
```

Expected: FAIL

- [ ] **Step 3: 実装**

```go
// internal/usecase/score/score.go
package score

import (
	"fmt"
	"math"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

// スコア配分: 潮汐(40) + 天気(35) + 時間帯(25) = 100
const (
	maxTideScore    = 40
	maxWeatherScore = 35
	maxTimeScore    = 25
)

type ScoreUsecase struct{}

func NewScoreUsecase() *ScoreUsecase {
	return &ScoreUsecase{}
}

func (u *ScoreUsecase) Calculate(weather *entity.WeatherData, tide *entity.TideData, now time.Time) *entity.FishingScore {
	tideScore := u.calculateTideScore(tide, now)
	weatherScore := u.calculateWeatherScore(weather)
	timeScore := u.calculateTimeScore(now)

	total := tideScore + weatherScore + timeScore
	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}

	return &entity.FishingScore{
		Total:        total,
		Rank:         entity.GetScoreRank(total),
		TideScore:    tideScore,
		WeatherScore: weatherScore,
		TimeScore:    timeScore,
		Explanation:  u.generateExplanation(tideScore, weatherScore, timeScore, tide, weather),
	}
}

// calculateTideScore: 満潮・干潮前後2時間が高スコア。大潮・中潮はボーナス。
func (u *ScoreUsecase) calculateTideScore(tide *entity.TideData, now time.Time) int {
	if tide == nil {
		return maxTideScore / 2
	}

	minMinutes := math.MaxFloat64
	allEvents := append(tide.HighTides, tide.LowTides...)
	for _, event := range allEvents {
		diff := math.Abs(now.Sub(event.Time).Minutes())
		if diff < minMinutes {
			minMinutes = diff
		}
	}

	var baseScore int
	switch {
	case minMinutes <= 30:
		baseScore = 40
	case minMinutes <= 60:
		baseScore = 35
	case minMinutes <= 90:
		baseScore = 28
	case minMinutes <= 120:
		baseScore = 20
	default:
		baseScore = 10
	}

	bonus := 0
	switch tide.TideType {
	case "大潮":
		bonus = 5
	case "中潮":
		bonus = 3
	}

	result := baseScore + bonus
	if result > maxTideScore {
		result = maxTideScore
	}
	return result
}

// calculateWeatherScore: 風速・気圧の安定性で評価。
func (u *ScoreUsecase) calculateWeatherScore(weather *entity.WeatherData) int {
	if weather == nil {
		return maxWeatherScore / 2
	}

	score := maxWeatherScore

	switch {
	case weather.WindSpeed > 10:
		score -= 20
	case weather.WindSpeed > 7:
		score -= 12
	case weather.WindSpeed > 5:
		score -= 6
	case weather.WindSpeed > 3:
		score -= 2
	}

	pressureDiff := math.Abs(weather.Pressure - 1013.0)
	switch {
	case pressureDiff > 15:
		score -= 8
	case pressureDiff > 10:
		score -= 4
	}

	if score < 0 {
		score = 0
	}
	return score
}

// calculateTimeScore: 早朝(4-7時)・夕方(16-19時)が高スコア。
func (u *ScoreUsecase) calculateTimeScore(now time.Time) int {
	hour := now.Hour()
	switch {
	case hour >= 4 && hour <= 7:
		return 25
	case hour >= 16 && hour <= 19:
		return 22
	case hour >= 8 && hour <= 10:
		return 18
	case hour >= 14 && hour <= 15:
		return 15
	case hour >= 20 && hour <= 22:
		return 12
	default:
		return 8
	}
}

func (u *ScoreUsecase) generateExplanation(tideScore, weatherScore, timeScore int, tide *entity.TideData, weather *entity.WeatherData) string {
	explanation := fmt.Sprintf("釣りやすさスコア：潮汐%d点＋天気%d点＋時間帯%d点", tideScore, weatherScore, timeScore)

	if tide != nil && tide.TideType != "" {
		explanation += fmt.Sprintf("。潮回り：%s", tide.TideType)
	}
	if weather != nil {
		explanation += fmt.Sprintf("。風速：%.1fm/s", weather.WindSpeed)
	}

	return explanation
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/usecase/score/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/usecase/score/
git commit -m "feat: 釣りやすさスコア算出ユースケース実装"
```

---

## Task 8: 場所検索ユースケース

**Files:**
- Create: `internal/usecase/location/location.go`
- Create: `internal/usecase/location/location_test.go`

**注意:** `MapsAPI` インターフェースは `[]*entity.LocationSearchResult` を返す。`infrastructure/external` を import しない。

- [ ] **Step 1: テストを書く**

```go
// internal/usecase/location/location_test.go
package location

import (
	"context"
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockMapsAPI struct{ mock.Mock }

func (m *MockMapsAPI) SearchLocations(query string, limit int) ([]*entity.LocationSearchResult, error) {
	args := m.Called(query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.LocationSearchResult), args.Error(1)
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
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func TestLocationUsecase_Search_CacheMiss(t *testing.T) {
	mockAPI := &MockMapsAPI{}
	mockCache := &MockCache{}

	results := []*entity.LocationSearchResult{
		{Name: "東京湾", Latitude: 35.5, Longitude: 139.8, Prefecture: "東京都"},
	}
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, 24*time.Hour).Return(nil)
	mockAPI.On("SearchLocations", "東京湾", 5).Return(results, nil)

	uc := NewLocationUsecase(mockAPI, mockCache)
	got, err := uc.Search(context.Background(), "東京湾", 5)

	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "東京湾", got[0].Name)
	mockAPI.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestLocationUsecase_Search_CacheHit(t *testing.T) {
	mockAPI := &MockMapsAPI{}
	mockCache := &MockCache{}

	cachedJSON := []byte(`[{"Name":"東京湾","Latitude":35.5,"Longitude":139.8,"Prefecture":"東京都","Region":""}]`)
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(cachedJSON, nil)

	uc := NewLocationUsecase(mockAPI, mockCache)
	got, err := uc.Search(context.Background(), "東京湾", 5)

	require.NoError(t, err)
	assert.Len(t, got, 1)
	mockAPI.AssertNotCalled(t, "SearchLocations")
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/usecase/location/... -v
```

Expected: FAIL

- [ ] **Step 3: 実装**

```go
// internal/usecase/location/location.go
package location

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type MapsAPI interface {
	SearchLocations(query string, limit int) ([]*entity.LocationSearchResult, error)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

const locationCacheTTL = 24 * time.Hour

type LocationUsecase struct {
	mapsAPI MapsAPI
	cache   Cache
}

func NewLocationUsecase(mapsAPI MapsAPI, cache Cache) *LocationUsecase {
	return &LocationUsecase{mapsAPI: mapsAPI, cache: cache}
}

func (u *LocationUsecase) Search(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error) {
	key := fmt.Sprintf("location:%s:%d", query, limit)

	if cached, _ := u.cache.Get(ctx, key); cached != nil {
		var results []*entity.LocationSearchResult
		if err := json.Unmarshal(cached, &results); err == nil {
			return results, nil
		}
	}

	results, err := u.mapsAPI.SearchLocations(query, limit)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(results); err == nil {
		u.cache.Set(ctx, key, b, locationCacheTTL)
	}

	return results, nil
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/usecase/location/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/usecase/location/
git commit -m "feat: 場所検索ユースケース実装"
```

---

## Task 9: FavoriteRepository + お気に入りユースケース

**Files:**
- Create: `internal/infrastructure/db/favorite_repository.go`
- Create: `internal/usecase/favorite/favorite.go`
- Create: `internal/usecase/favorite/favorite_test.go`

**注意:**
- `*sql.DB` を使う（`pgxpool.Pool` ではない）
- `toLocationEntity` で `sql.NullString`・`uuid.NullUUID` の null チェックが必要
- `sqlcgen.LocationType` → `entity.LocationType` へのキャスト

- [ ] **Step 1: `favorite_repository.go` を実装する**

```go
// internal/infrastructure/db/favorite_repository.go
package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	sqlcgen "github.com/kazumadev619-dev/fishing-api/db/generated"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
)

type favoriteRepository struct {
	queries *sqlcgen.Queries
}

func NewFavoriteRepository(db *sql.DB) repository.FavoriteRepository {
	return &favoriteRepository{queries: sqlcgen.New(db)}
}

func (r *favoriteRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	rows, err := r.queries.FindFavoritesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	locations := make([]*entity.Location, 0, len(rows))
	for _, row := range rows {
		locations = append(locations, toLocationEntity(row))
	}
	return locations, nil
}

func (r *favoriteRepository) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	return r.queries.AddFavorite(ctx, sqlcgen.AddFavoriteParams{
		ID:         uuid.New(),
		UserID:     userID,
		LocationID: locationID,
	})
}

func (r *favoriteRepository) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	return r.queries.DeleteFavorite(ctx, sqlcgen.DeleteFavoriteParams{
		UserID:     userID,
		LocationID: locationID,
	})
}

func (r *favoriteRepository) Exists(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) (bool, error) {
	return r.queries.FavoriteExists(ctx, sqlcgen.FavoriteExistsParams{
		UserID:     userID,
		LocationID: locationID,
	})
}

func toLocationEntity(row sqlcgen.Location) *entity.Location {
	loc := &entity.Location{
		ID:           row.ID,
		Name:         row.Name,
		Latitude:     row.Latitude,
		Longitude:    row.Longitude,
		LocationType: entity.LocationType(row.LocationType),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	if row.Region.Valid {
		loc.Region = &row.Region.String
	}
	if row.Prefecture.Valid {
		loc.Prefecture = &row.Prefecture.String
	}
	if row.PortID.Valid {
		loc.PortID = &row.PortID.UUID
	}
	return loc
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/infrastructure/db/...
```

Expected: エラーなし

- [ ] **Step 3: テストを書く**

```go
// internal/usecase/favorite/favorite_test.go
package favorite

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockFavoriteRepo struct{ mock.Mock }

func (m *MockFavoriteRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Location), args.Error(1)
}
func (m *MockFavoriteRepo) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	args := m.Called(ctx, userID, locationID)
	return args.Error(0)
}
func (m *MockFavoriteRepo) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	args := m.Called(ctx, userID, locationID)
	return args.Error(0)
}
func (m *MockFavoriteRepo) Exists(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, locationID)
	return args.Bool(0), args.Error(1)
}

func TestFavoriteUsecase_GetList(t *testing.T) {
	repo := &MockFavoriteRepo{}
	userID := uuid.New()
	name := "テスト釣り場"
	locations := []*entity.Location{{ID: uuid.New(), Name: name}}

	repo.On("FindByUserID", mock.Anything, userID).Return(locations, nil)

	uc := NewFavoriteUsecase(repo)
	result, err := uc.GetList(context.Background(), userID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "テスト釣り場", result[0].Name)
	repo.AssertExpectations(t)
}

func TestFavoriteUsecase_Add_Success(t *testing.T) {
	repo := &MockFavoriteRepo{}
	userID := uuid.New()
	locationID := uuid.New()

	repo.On("Exists", mock.Anything, userID, locationID).Return(false, nil)
	repo.On("Add", mock.Anything, userID, locationID).Return(nil)

	uc := NewFavoriteUsecase(repo)
	err := uc.Add(context.Background(), userID, locationID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestFavoriteUsecase_Add_AlreadyExists(t *testing.T) {
	repo := &MockFavoriteRepo{}
	userID := uuid.New()
	locationID := uuid.New()

	repo.On("Exists", mock.Anything, userID, locationID).Return(true, nil)

	uc := NewFavoriteUsecase(repo)
	err := uc.Add(context.Background(), userID, locationID)
	assert.Error(t, err)
	repo.AssertNotCalled(t, "Add")
}

func TestFavoriteUsecase_Delete(t *testing.T) {
	repo := &MockFavoriteRepo{}
	userID := uuid.New()
	locationID := uuid.New()

	repo.On("Delete", mock.Anything, userID, locationID).Return(nil)

	uc := NewFavoriteUsecase(repo)
	err := uc.Delete(context.Background(), userID, locationID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
```

- [ ] **Step 4: テストが失敗することを確認**

```bash
go test ./internal/usecase/favorite/... -v
```

Expected: FAIL

- [ ] **Step 5: `favorite.go` を実装する**

```go
// internal/usecase/favorite/favorite.go
package favorite

import (
	"context"
	"errors"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
)

type FavoriteUsecase struct {
	repo repository.FavoriteRepository
}

func NewFavoriteUsecase(repo repository.FavoriteRepository) *FavoriteUsecase {
	return &FavoriteUsecase{repo: repo}
}

func (u *FavoriteUsecase) GetList(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	return u.repo.FindByUserID(ctx, userID)
}

func (u *FavoriteUsecase) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	exists, err := u.repo.Exists(ctx, userID, locationID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrAlreadyExists
	}
	return u.repo.Add(ctx, userID, locationID)
}

func (u *FavoriteUsecase) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	_ = errors.New // errors パッケージを確実に使う
	return u.repo.Delete(ctx, userID, locationID)
}
```

**注意:** `Delete` の `errors` import は不要なので削除した最終版：

```go
// internal/usecase/favorite/favorite.go
package favorite

import (
	"context"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
)

type FavoriteUsecase struct {
	repo repository.FavoriteRepository
}

func NewFavoriteUsecase(repo repository.FavoriteRepository) *FavoriteUsecase {
	return &FavoriteUsecase{repo: repo}
}

func (u *FavoriteUsecase) GetList(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	return u.repo.FindByUserID(ctx, userID)
}

func (u *FavoriteUsecase) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	exists, err := u.repo.Exists(ctx, userID, locationID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrAlreadyExists
	}
	return u.repo.Add(ctx, userID, locationID)
}

func (u *FavoriteUsecase) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	return u.repo.Delete(ctx, userID, locationID)
}
```

- [ ] **Step 6: テストが通ることを確認**

```bash
go test ./internal/usecase/favorite/... -v
```

Expected: PASS

- [ ] **Step 7: コミット**

```bash
git add internal/infrastructure/db/favorite_repository.go internal/usecase/favorite/
git commit -m "feat: FavoriteRepository・お気に入りユースケース実装"
```

---

## Task 10: 全ハンドラー・ルーター・main.go 更新

**Files:**
- Create: `internal/interface/handler/weather_handler.go`
- Create: `internal/interface/handler/tide_handler.go`
- Create: `internal/interface/handler/location_handler.go`
- Create: `internal/interface/handler/favorite_handler.go`
- Modify: `internal/interface/router/router.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: `weather_handler.go` を実装する**

```go
// internal/interface/handler/weather_handler.go
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/pkg/validator"
)

type WeatherUsecaseInterface interface {
	GetCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error)
	GetForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error)
}

type WeatherHandler struct {
	usecase WeatherUsecaseInterface
}

func NewWeatherHandler(uc WeatherUsecaseInterface) *WeatherHandler {
	return &WeatherHandler{usecase: uc}
}

func (h *WeatherHandler) Get(c *gin.Context) {
	lat, lon, err := validator.ParseAndValidateCoordinates(
		c.Query("lat"), c.Query("lon"),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_PARAMS", "status": 400})
		return
	}

	weatherType := c.DefaultQuery("type", "current")
	ctx := c.Request.Context()

	switch weatherType {
	case "current":
		data, err := h.usecase.GetCurrent(ctx, lat, lon)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch weather", "code": "INTERNAL_ERROR", "status": 500})
			return
		}
		c.JSON(http.StatusOK, data)
	case "forecast":
		data, err := h.usecase.GetForecast(ctx, lat, lon)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch forecast", "code": "INTERNAL_ERROR", "status": 500})
			return
		}
		c.JSON(http.StatusOK, data)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'current' or 'forecast'", "code": "INVALID_PARAMS", "status": 400})
	}
}
```

- [ ] **Step 2: `tide_handler.go` を実装する**

```go
// internal/interface/handler/tide_handler.go
package handler

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type TideUsecaseInterface interface {
	GetTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error)
}

type TideHandler struct {
	usecase TideUsecaseInterface
}

func NewTideHandler(uc TideUsecaseInterface) *TideHandler {
	return &TideHandler{usecase: uc}
}

var (
	prefCodeRegex = regexp.MustCompile(`^[0-9]{1,2}$`)
	portCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	dateRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func (h *TideHandler) Get(c *gin.Context) {
	prefCode := c.Query("prefectureCode")
	portCode := c.Query("portCode")
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	if prefCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prefectureCode is required", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if portCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "portCode is required", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if !prefCodeRegex.MatchString(prefCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prefectureCode format", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if !portCodeRegex.MatchString(portCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portCode format", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if !dateRegex.MatchString(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format. Use YYYY-MM-DD", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	data, err := h.usecase.GetTideData(c.Request.Context(), prefCode, portCode, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tide data", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, data)
}
```

- [ ] **Step 3: `location_handler.go` を実装する**

**注意:** `entity.LocationSearchResult` を使う。`infrastructure/external` は import しない。

```go
// internal/interface/handler/location_handler.go
package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type LocationUsecaseInterface interface {
	Search(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error)
}

type LocationHandler struct {
	usecase LocationUsecaseInterface
}

func NewLocationHandler(uc LocationUsecaseInterface) *LocationHandler {
	return &LocationHandler{usecase: uc}
}

func (h *LocationHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query must be at least 2 characters", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if len(query) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query must be less than 200 characters", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter", "code": "INVALID_PARAMS", "status": 400})
			return
		}
		limit = parsed
	}

	results, err := h.usecase.Search(c.Request.Context(), query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "location search failed", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}
```

- [ ] **Step 4: `favorite_handler.go` を実装する**

**注意:** `errors.Is` を使う（`==` 禁止）。

```go
// internal/interface/handler/favorite_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type FavoriteUsecaseInterface interface {
	GetList(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error)
	Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
	Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
}

type FavoriteHandler struct {
	usecase FavoriteUsecaseInterface
}

func NewFavoriteHandler(uc FavoriteUsecaseInterface) *FavoriteHandler {
	return &FavoriteHandler{usecase: uc}
}

func (h *FavoriteHandler) GetList(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	locations, err := h.usecase.GetList(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get favorites", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorites": locations})
}

type addFavoriteRequest struct {
	LocationID string `json:"location_id" binding:"required"`
}

func (h *FavoriteHandler) Add(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req addFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_PARAMS", "status": 400})
		return
	}

	locationID, err := uuid.Parse(req.LocationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid location_id", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	if err := h.usecase.Add(c.Request.Context(), userID, locationID); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "already favorited", "code": "ALREADY_EXISTS", "status": 409})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add favorite", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "added to favorites"})
}

func (h *FavoriteHandler) Delete(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	locationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid location id", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), userID, locationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete favorite", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed from favorites"})
}
```

- [ ] **Step 5: `router.go` を更新する**

**注意:**
- import は `jwtutil "github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"`（`pkg/jwt` ではない）
- `New` は `*jwtutil.Manager` を受け取る

```go
// internal/interface/router/router.go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/middleware"
	"github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	Weather  *handler.WeatherHandler
	Tide     *handler.TideHandler
	Location *handler.LocationHandler
	Favorite *handler.FavoriteHandler
}

func New(handlers *Handlers, jwtManager *jwtutil.Manager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/health", handler.HealthCheck)

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", handlers.Auth.Register)
			authGroup.POST("/login", handlers.Auth.Login)
			authGroup.POST("/refresh", handlers.Auth.RefreshToken)
			authGroup.GET("/verify-email", handlers.Auth.VerifyEmail)
		}

		// 天気・潮汐・場所検索（認証不要）
		api.GET("/weather", handlers.Weather.Get)
		api.GET("/conditions/tide", handlers.Tide.Get)
		api.GET("/locations/search", handlers.Location.Search)

		// 認証必要ルート
		protected := api.Group("").Use(middleware.JWTAuth(jwtManager))
		{
			protected.GET("/favorites", handlers.Favorite.GetList)
			protected.POST("/favorites", handlers.Favorite.Add)
			protected.DELETE("/favorites/:id", handlers.Favorite.Delete)
		}
	}

	return r
}
```

- [ ] **Step 6: `main.go` を更新する（slog・jwtManagerAdapter を維持）**

**注意:**
- `slog.Error` + `os.Exit(1)` を維持
- `jwtManagerAdapter` は認証ユースケース用に維持
- `router.New(handlers, jwtManager)` に `*jwtutil.Manager` を直接渡す

```go
// cmd/server/main.go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kazumadev619-dev/fishing-api/config"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/cache"
	infradb "github.com/kazumadev619-dev/fishing-api/internal/infrastructure/db"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/email"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/external"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/router"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/auth"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/favorite"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/location"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/score"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/tide"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/weather"
	"github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
)

// jwtManagerAdapter は *jwtutil.Manager を auth.JWTManager インターフェースに適合させる
type jwtManagerAdapter struct{ m *jwtutil.Manager }

func (a *jwtManagerAdapter) GenerateAccessToken(userID uuid.UUID) (string, error) {
	return a.m.GenerateAccessToken(userID)
}

func (a *jwtManagerAdapter) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	return a.m.GenerateRefreshToken(userID)
}

func (a *jwtManagerAdapter) ValidateRefreshToken(tokenStr string) (uuid.UUID, error) {
	claims, err := a.m.ValidateRefreshToken(tokenStr)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := infradb.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()

	cacheClient, err := cache.NewCacheClient(ctx, cfg.Redis.URL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	// JWT
	jwtManager := jwtutil.NewManager(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret)

	// Repositories
	userRepo := infradb.NewUserRepository(db)
	tokenRepo := infradb.NewVerificationTokenRepository(db)
	favoriteRepo := infradb.NewFavoriteRepository(db)

	// External clients
	weatherClient := external.NewWeatherClient(cfg.External.OpenWeatherAPIKey)
	tideClient := external.NewTideClient()
	mapsClient := external.NewMapsClient(cfg.External.GoogleMapsAPIKey)
	emailClient := email.NewEmailClient(cfg.Email.ResendAPIKey, cfg.Email.FromAddress)

	// Usecases
	authUC := auth.NewAuthUsecase(userRepo, tokenRepo, emailClient, &jwtManagerAdapter{m: jwtManager}, cfg.Server.AppBaseURL)
	weatherUC := weather.NewWeatherUsecase(weatherClient, cacheClient)
	tideUC := tide.NewTideUsecase(tideClient, cacheClient)
	locationUC := location.NewLocationUsecase(mapsClient, cacheClient)
	scoreUC := score.NewScoreUsecase()
	_ = scoreUC // ハンドラー実装時に使用（スコアAPIは天気・潮汐と組み合わせて計算）
	favoriteUC := favorite.NewFavoriteUsecase(favoriteRepo)

	// Handlers
	handlers := &router.Handlers{
		Auth:     handler.NewAuthHandler(authUC),
		Weather:  handler.NewWeatherHandler(weatherUC),
		Tide:     handler.NewTideHandler(tideUC),
		Location: handler.NewLocationHandler(locationUC),
		Favorite: handler.NewFavoriteHandler(favoriteUC),
	}

	r := router.New(handlers, jwtManager)

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 7: ビルド確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 8: 全テストが通ることを確認**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: 全パッケージ `ok`

- [ ] **Step 9: コミット**

```bash
git add internal/interface/handler/weather_handler.go \
        internal/interface/handler/tide_handler.go \
        internal/interface/handler/location_handler.go \
        internal/interface/handler/favorite_handler.go \
        internal/interface/router/router.go \
        cmd/server/main.go
git commit -m "feat: 全APIハンドラー・ルーター実装、Phase 3完了"
```

---

## 完了条件チェックリスト

- [ ] `go build ./...` エラーなし
- [ ] `go test ./...` 全テスト PASS
- [ ] `GET /api/weather?lat=35.6895&lon=139.6917&type=current` が天気データを返す
- [ ] `GET /api/conditions/tide?prefectureCode=13&portCode=TK` が潮汐データを返す
- [ ] `GET /api/locations/search?q=東京湾` が場所リストを返す
- [ ] `GET /api/favorites` が認証なしで 401 を返す
- [ ] JWTトークン付きで `GET /api/favorites` がお気に入りリストを返す
- [ ] `POST /api/favorites` でお気に入り追加、`DELETE /api/favorites/:id` で削除できる
