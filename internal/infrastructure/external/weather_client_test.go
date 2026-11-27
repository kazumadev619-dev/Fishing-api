package external

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 契約テスト：testdata/owm_*.json は OpenWeatherMap の公式ドキュメントの応答例（Zocca の例）。
//   - owm_current.json:  https://openweathermap.org/current の JSON の例
//   - owm_forecast.json: https://openweathermap.org/forecast5 の JSON の例（list を2件に縮めた）
//
// 本物の応答はキーが要るので保存していない。キーも個人情報も入っていない。
func serveOWM(t *testing.T, path, fixture string, gotQuery *url.Values) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile("testdata/" + fixture)
	require.NoError(t, err)
	return serveOWMBody(t, path, body, gotQuery)
}

func serveOWMBody(t *testing.T, path string, body []byte, gotQuery *url.Values) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		if gotQuery != nil {
			*gotQuery = r.URL.Query()
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestWeatherClient_FetchCurrent_Contract(t *testing.T) {
	var q url.Values
	srv := serveOWM(t, "/data/2.5/weather", "owm_current.json", &q)
	c := newWeatherClientWithHTTPClient("test-key", srv.URL, srv.Client())

	w, err := c.FetchCurrent(t.Context(), 44.34, 10.99, "standard")
	require.NoError(t, err)

	assert.Equal(t, "44.34", q.Get("lat"))
	assert.Equal(t, "10.99", q.Get("lon"))
	assert.Equal(t, "standard", q.Get("units"))
	assert.Equal(t, "ja", q.Get("lang"))
	assert.Equal(t, "test-key", q.Get("appid"))

	assert.InDelta(t, 44.34, w.Lat, 1e-9)
	assert.InDelta(t, 10.99, w.Lon, 1e-9)
	assert.Equal(t, "Zocca", w.CityName)
	assert.Equal(t, "IT", w.Country)
	assert.InDelta(t, 298.48, w.Temperature, 1e-9)
	assert.InDelta(t, 298.74, w.FeelsLike, 1e-9)
	assert.InDelta(t, 297.56, w.TempMin, 1e-9)
	assert.InDelta(t, 300.05, w.TempMax, 1e-9)
	assert.Equal(t, 64, w.Humidity)
	assert.InDelta(t, 1015, w.Pressure, 1e-9)
	assert.InDelta(t, 0.62, w.WindSpeed, 1e-9)
	assert.Equal(t, 349, w.WindDeg)
	require.NotNil(t, w.WindGust)
	assert.InDelta(t, 1.18, *w.WindGust, 1e-9)
	assert.Equal(t, 100, w.Cloudiness)
	assert.Equal(t, 10000, w.Visibility)
	assert.Equal(t, 501, w.Condition.ID)
	assert.Equal(t, "Rain", w.Condition.Main)
	assert.Equal(t, "moderate rain", w.Condition.Description)
	assert.Equal(t, "10d", w.Condition.Icon)
	require.NotNil(t, w.Rain)
	assert.InDelta(t, 3.16, *w.Rain, 1e-9)
	assert.Nil(t, w.Snow)
	assert.True(t, w.Sunrise.Equal(time.Unix(1661834187, 0)))
	assert.True(t, w.Sunset.Equal(time.Unix(1661882248, 0)))
	assert.True(t, w.DateTime.Equal(time.Unix(1661870592, 0)))
	assert.Equal(t, 7200, w.Timezone)
}

// 本物の例から1項目だけ差し替えた応答を作る。
func owmCurrentWith(t *testing.T, mutate func(m map[string]any)) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/owm_current.json")
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(body, &m))
	mutate(m)
	out, err := json.Marshal(m)
	require.NoError(t, err)
	return out
}

// #414 相当：風速・気圧が数でない・欠けている応答は 502 の上流の失敗にする。
func TestWeatherClient_FetchCurrent_InvalidValues(t *testing.T) {
	wind := func(m map[string]any) map[string]any { return m["wind"].(map[string]any) }
	main := func(m map[string]any) map[string]any { return m["main"].(map[string]any) }
	tests := []struct {
		name   string
		mutate func(m map[string]any)
	}{
		{"speed が無い", func(m map[string]any) { delete(wind(m), "speed") }},
		{"speed が null", func(m map[string]any) { wind(m)["speed"] = nil }},
		{"speed が文字列", func(m map[string]any) { wind(m)["speed"] = "5" }},
		{"speed が true", func(m map[string]any) { wind(m)["speed"] = true }},
		{"wind 自体が無い", func(m map[string]any) { delete(m, "wind") }},
		{"wind が null", func(m map[string]any) { m["wind"] = nil }},
		{"pressure が無い", func(m map[string]any) { delete(main(m), "pressure") }},
		{"pressure が null", func(m map[string]any) { main(m)["pressure"] = nil }},
		{"pressure が文字列", func(m map[string]any) { main(m)["pressure"] = "1013" }},
		{"pressure が配列", func(m map[string]any) { main(m)["pressure"] = []int{1013} }},
		{"main 自体が無い", func(m map[string]any) { delete(m, "main") }},
		{"weather が空", func(m map[string]any) { m["weather"] = []any{} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := serveOWMBody(t, "/data/2.5/weather", owmCurrentWith(t, tt.mutate), nil)
			c := newWeatherClientWithHTTPClient("k", srv.URL, srv.Client())

			_, err := c.FetchCurrent(t.Context(), 35, 139, "metric")

			var ue *domain.UpstreamError
			require.ErrorAs(t, err, &ue)
			assert.Equal(t, domain.UpstreamServerError, ue.Kind)
			assert.Equal(t, 502, ue.StatusCode)
		})
	}
}

// 0 は欠けではない。
func TestWeatherClient_FetchCurrent_ZeroWindIsValid(t *testing.T) {
	body := owmCurrentWith(t, func(m map[string]any) { m["wind"].(map[string]any)["speed"] = 0 })
	srv := serveOWMBody(t, "/data/2.5/weather", body, nil)
	c := newWeatherClientWithHTTPClient("k", srv.URL, srv.Client())

	w, err := c.FetchCurrent(t.Context(), 35, 139, "metric")
	require.NoError(t, err)
	assert.Zero(t, w.WindSpeed)
}

// 予報は Next と同じく上流の応答をそのまま返す。
func TestWeatherClient_FetchForecast_Contract(t *testing.T) {
	var q url.Values
	srv := serveOWM(t, "/data/2.5/forecast", "owm_forecast.json", &q)
	c := newWeatherClientWithHTTPClient("k", srv.URL, srv.Client())

	raw, err := c.FetchForecast(t.Context(), 44.34, 10.99, "imperial")
	require.NoError(t, err)
	assert.Equal(t, "imperial", q.Get("units"))

	want, err := os.ReadFile("testdata/owm_forecast.json")
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(raw))
}

func TestWeatherClient_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := newWeatherClientWithHTTPClient("k", srv.URL, srv.Client())

	_, err := c.FetchCurrent(t.Context(), 35, 139, "metric")
	var ue *domain.UpstreamError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, domain.UpstreamClientError, ue.Kind)
	assert.NotContains(t, err.Error(), "appid", "エラーに API キー入りの URL を含めない")

	_, err = c.FetchForecast(t.Context(), 35, 139, "metric")
	require.ErrorAs(t, err, &ue)
}

// キーが未設定なら上流を呼ばずに ErrUpstreamNotConfigured を返す（Next の 503 API_NOT_CONFIGURED）。
func TestWeatherClient_NotConfigured(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	c := newWeatherClientWithHTTPClient("", srv.URL, srv.Client())

	_, err := c.FetchCurrent(t.Context(), 35, 139, "metric")
	assert.ErrorIs(t, err, domain.ErrUpstreamNotConfigured)
	_, err = c.FetchForecast(t.Context(), 35, 139, "metric")
	assert.ErrorIs(t, err, domain.ErrUpstreamNotConfigured)
	assert.False(t, called)
}
