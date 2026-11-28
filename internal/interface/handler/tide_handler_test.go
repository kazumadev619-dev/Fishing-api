package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTideUsecase struct{ mock.Mock }

func (m *MockTideUsecase) GetTideData(ctx context.Context, prefCode, portCode, date, tideRange string) (*entity.TideData, bool, error) {
	args := m.Called(ctx, prefCode, portCode, date, tideRange)
	if args.Get(0) == nil {
		return nil, false, args.Error(2)
	}
	return args.Get(0).(*entity.TideData), args.Bool(1), args.Error(2)
}

func serveTide(m *MockTideUsecase, url string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterOpenAPI(r, nil, NewTideHandler(m))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequestWithContext(context.Background(), http.MethodGet, url, nil))
	return w
}

// 応答の JSON は Next の /api/conditions/tide と同じキー（data.port / data.tides[].daily / meta）。
func TestTideHandler_Get_ResponseShape(t *testing.T) {
	name := "満月"
	data := &entity.TideData{
		Port: entity.TidePort{PrefectureCode: "13", PortCode: "2", PortName: "芝浦"},
		Days: []entity.DailyTide{{
			Date:  "2026-11-27",
			Sun:   entity.SunInfo{AstroTwilight: []string{"04:58", "17:59"}, RegularTwilight: []string{"06:00", "16:57"}, Rise: "06:28", Midline: "11:29", Set: "16:29"},
			Moon:  entity.MoonInfo{Brightness: "90.9", Age: "17.8", Title: "中潮", Rise: "26日 17:46", Midline: "27日 01:34", Set: "27日 09:19", Name: &name},
			Tide:  []entity.TideEvent{{Time: "00:00", UnixMs: 1795705200000, CM: -13.6}},
			Flood: []entity.TideEvent{{Time: "07:26", UnixMs: 1795731960000, CM: 197.7}},
			Ebb:   []entity.TideEvent{{Time: "00:13", UnixMs: 1795705980000, CM: -14.2}},
		}},
		DataTime: time.Date(2026, 11, 27, 1, 2, 3, 4_000_000, time.UTC),
	}
	m := &MockTideUsecase{}
	m.On("GetTideData", mock.Anything, "13", "2", "2026-11-27", "day").Return(data, true, nil)

	w := serveTide(m, "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27&range=day")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "private, max-age=3600", w.Header().Get("Cache-Control"))
	body := w.Body.String()
	assert.Regexp(t, `"fetchedAt":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z"`, body)
	assert.JSONEq(t, `{
		"status": 200,
		"data": {
			"port": {"prefecture_code": "13", "port_code": "2", "port_name": "芝浦"},
			"tides": [{
				"date": "2026-11-27",
				"daily": {
					"sun": {"astro_twilight": ["04:58", "17:59"], "regular_twilight": ["06:00", "16:57"], "rise": "06:28", "midline": "11:29", "set": "16:29"},
					"moon": {"brightness": "90.9", "age": "17.8", "title": "中潮", "rise": "26日 17:46", "midline": "27日 01:34", "set": "27日 09:19", "name": "満月"},
					"tide": [{"time": "00:00", "unix": 1795705200000, "cm": -13.6}],
					"flood": [{"time": "07:26", "unix": 1795731960000, "cm": 197.7}],
					"edd": [{"time": "00:13", "unix": 1795705980000, "cm": -14.2}]
				}
			}],
			"dataTime": "2026-11-27T01:02:03.004Z"
		},
		"meta": {"fromCache": true, "fetchedAt": "IGNORED"}
	}`, replaceFetchedAt(body))
	m.AssertExpectations(t)
}

// 既定値：range は week、date は JST の今日。キャッシュに無ければ max-age=300。
func TestTideHandler_Get_Defaults(t *testing.T) {
	today := time.Now().In(time.FixedZone("JST", 9*60*60)).Format("2006-01-02")
	m := &MockTideUsecase{}
	m.On("GetTideData", mock.Anything, "13", "2", today, "week").Return(&entity.TideData{}, false, nil)

	w := serveTide(m, "/api/conditions/tide?prefectureCode=13&portCode=2")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "private, max-age=300", w.Header().Get("Cache-Control"))
	assert.Contains(t, w.Body.String(), `"tides":[]`, "日が無くても null でなく空の配列")
	m.AssertExpectations(t)
}

func TestTideHandler_Get_Errors(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		ucErr      error
		wantStatus int
		wantBody   string
	}{
		{
			"prefectureCode なし", "/api/conditions/tide?portCode=2", nil, 400,
			`{"error":"Missing required parameter: prefectureCode","status":400}`,
		},
		{
			"portCode なし", "/api/conditions/tide?prefectureCode=13", nil, 400,
			`{"error":"Missing required parameter: portCode","status":400}`,
		},
		{
			"prefectureCode の形が不正", "/api/conditions/tide?prefectureCode=abc&portCode=2", nil, 400,
			`{"error":"Invalid prefectureCode format","status":400}`,
		},
		{
			"portCode の形が不正", "/api/conditions/tide?prefectureCode=13&portCode=a-b", nil, 400,
			`{"error":"Invalid portCode format","status":400}`,
		},
		{
			"存在しない日付", "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-02-30", nil, 400,
			`{"error":"Invalid date. Use an existing date in YYYY-MM-DD format","status":400}`,
		},
		{
			"range が不正", "/api/conditions/tide?prefectureCode=13&portCode=2&range=year", nil, 400,
			`{"error":"Invalid range. Must be one of: day, week, month","status":400}`,
		},
		{
			"上流の失敗は 502", "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27",
			domain.NewInvalidUpstreamResponse("status 0"), 502,
			`{"error":"サーバーで問題が発生しました。サポートに連絡してください。","code":"SERVER_ERROR","retryable":true}`,
		},
		{
			"上流のタイムアウトは 504", "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27",
			&domain.UpstreamError{Kind: domain.UpstreamTimeout, Retryable: true}, 504,
			`{"error":"通信がタイムアウトしました。インターネット接続を確認して、もう一度お試しください。","code":"TIMEOUT","retryable":true}`,
		},
		{
			"想定外は 500", "/api/conditions/tide?prefectureCode=13&portCode=2&date=2026-11-27",
			errors.New("boom"), 500, `{"error":"Failed to fetch tide data","status":500}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MockTideUsecase{}
			if tt.ucErr != nil {
				m.On("GetTideData", mock.Anything, "13", "2", "2026-11-27", "week").Return(nil, false, tt.ucErr)
			}

			w := serveTide(m, tt.url)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.JSONEq(t, tt.wantBody, w.Body.String())
			m.AssertExpectations(t)
		})
	}
}

var fetchedAtRe = regexp.MustCompile(`"fetchedAt":"[^"]*"`)

// replaceFetchedAt は毎回変わる fetchedAt を比べられる固定値にする（形は別に確かめる）。
func replaceFetchedAt(body string) string {
	return fetchedAtRe.ReplaceAllString(body, `"fetchedAt":"IGNORED"`)
}
