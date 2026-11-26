package external

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 契約テスト：testdata/ は tide736.net の本物の応答（2026-11-27 に取得。芝浦、pc=13 hc=2）。
//   - tide736_week.json:    get_tide.php?pc=13&hc=2&yr=2026&mn=11&dy=27&rg=week
//   - tide736_status0.json: get_tide.php?pc=13&hc=2&yr=2026&mn=11&dy=27&rg=bogus（status 0）
//
// 取り直すときは、上の URL を curl で保存する（キーは不要）。
func serveFixture(t *testing.T, name string, gotQuery *url.Values) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/get_tide.php" {
			http.NotFound(w, r)
			return
		}
		if gotQuery != nil {
			*gotQuery = r.URL.Query()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTideClient_FetchTideData_Contract(t *testing.T) {
	var q url.Values
	srv := serveFixture(t, "tide736_week.json", &q)
	c := newTideClientWithHTTPClient(srv.URL, srv.Client())

	data, err := c.FetchTideData(t.Context(), "13", "2", "2026-11-27", "week")
	require.NoError(t, err)

	// Next と同じクエリで get_tide.php を呼ぶ
	assert.Equal(t, url.Values{
		"pc": {"13"}, "hc": {"2"}, "yr": {"2026"}, "mn": {"11"}, "dy": {"27"}, "rg": {"week"},
	}, q)

	// 港：数で返る prefecture_code / harbor_code を文字列にする（Next の String(...) と同じ）
	assert.Equal(t, "13", data.Port.PrefectureCode)
	assert.Equal(t, "2", data.Port.PortCode)
	assert.Equal(t, "芝浦", data.Port.PortName)

	// 日付の昇順に7日分
	require.Len(t, data.Days, 7)
	assert.Equal(t, "2026-11-27", data.Days[0].Date)
	assert.Equal(t, "2026-12-03", data.Days[6].Date)

	// 1日目の中身（値は保存した応答から読んだもの）
	d := data.Days[0]
	assert.Equal(t, "06:28", d.Sun.Rise)
	assert.Equal(t, "16:29", d.Sun.Set)
	assert.Equal(t, []string{"04:58", "17:59"}, d.Sun.AstroTwilight)
	assert.Equal(t, "中潮", d.Moon.Title)
	assert.Equal(t, "17.8", d.Moon.Age)
	assert.Nil(t, d.Moon.Name)
	require.Len(t, d.Flood, 2)
	assert.Equal(t, "07:26", d.Flood[0].Time)
	assert.Equal(t, int64(1795731960000), d.Flood[0].UnixMs)
	assert.InDelta(t, 197.7, d.Flood[0].CM, 1e-9)
	require.Len(t, d.Ebb, 2)
	assert.InDelta(t, -14.2, d.Ebb[0].CM, 1e-9)
	assert.Len(t, d.Tide, 73)
	assert.False(t, data.DataTime.IsZero())
}

// 上流が status 0（異常終了）を返したら、502 にする上流の失敗（SERVER_ERROR）。
func TestTideClient_FetchTideData_Status0(t *testing.T) {
	srv := serveFixture(t, "tide736_status0.json", nil)
	c := newTideClientWithHTTPClient(srv.URL, srv.Client())

	_, err := c.FetchTideData(t.Context(), "13", "2", "2026-11-27", "week")

	var ue *domain.UpstreamError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, domain.UpstreamServerError, ue.Kind)
	assert.Equal(t, 502, ue.StatusCode)
}

func TestTideClient_FetchTideData_UpstreamDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := newTideClientWithHTTPClient(srv.URL, srv.Client())

	_, err := c.FetchTideData(t.Context(), "13", "2", "2026-11-27", "week")

	var ue *domain.UpstreamError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, domain.UpstreamClientError, ue.Kind)
}
