package external

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

// TideClient は tide736.net の潮位 API（get_tide.php）のクライアント。
type TideClient struct {
	baseURL string
	fetch   *fetcher
}

// NewTideClient は本番用クライアントを返す。
func NewTideClient() *TideClient {
	return newTideClientWithHTTPClient("https://tide736.net", &http.Client{})
}

func newTideClientWithHTTPClient(baseURL string, httpClient *http.Client) *TideClient {
	return &TideClient{baseURL: baseURL, fetch: newFetcher(httpClient)}
}

// tide736Response は get_tide.php の応答。形は testdata/tide736_week.json を正とする。
type tide736Response struct {
	Status  int    `json:"status"` // 0: 異常終了、1: 正常終了
	Message string `json:"message"`
	Tide    struct {
		Port struct {
			PrefectureCode json.Number `json:"prefecture_code"`
			HarborCode     json.Number `json:"harbor_code"`
			HarborNameJ    string      `json:"harbor_namej"`
		} `json:"port"`
		Chart map[string]tide736Daily `json:"chart"`
	} `json:"tide"`
}

type tide736Daily struct {
	Sun struct {
		AstroTwilight   []string `json:"astro_twilight"`
		RegularTwilight []string `json:"regular_twilight"`
		Rise            string   `json:"rise"`
		Midline         string   `json:"midline"`
		Set             string   `json:"set"`
	} `json:"sun"`
	Moon struct {
		Brightness string  `json:"brightness"`
		Age        string  `json:"age"`
		Title      string  `json:"title"`
		Rise       string  `json:"rise"`
		Midline    string  `json:"midline"`
		Set        string  `json:"set"`
		Name       *string `json:"name"`
	} `json:"moon"`
	Tide  []tide736Event `json:"tide"`
	Flood []tide736Event `json:"flood"`
	Edd   []tide736Event `json:"edd"`
}

type tide736Event struct {
	Time string  `json:"time"`
	Unix int64   `json:"unix"`
	CM   float64 `json:"cm"`
}

// FetchTideData は date（YYYY-MM-DD）から rng（day / week / month）の範囲の潮汐を取得する。
func (c *TideClient) FetchTideData(ctx context.Context, prefCode, portCode, date, rng string) (*entity.TideData, error) {
	ymd := strings.Split(date, "-")
	if len(ymd) != 3 {
		return nil, fmt.Errorf("invalid date %q", date)
	}
	q := url.Values{
		"pc": {prefCode}, "hc": {portCode},
		"yr": {ymd[0]}, "mn": {ymd[1]}, "dy": {ymd[2]},
		"rg": {rng},
	}

	var raw tide736Response
	if err := c.fetch.getJSON(ctx, c.baseURL+"/api/get_tide.php?"+q.Encode(), &raw); err != nil {
		return nil, fmt.Errorf("tide736: %w", err)
	}
	if raw.Status != 1 {
		return nil, fmt.Errorf("tide736: %w", domain.NewInvalidUpstreamResponse("tide736 API error: %s", raw.Message))
	}
	return toTideData(&raw), nil
}

func toTideData(raw *tide736Response) *entity.TideData {
	dates := slices.Sorted(maps.Keys(raw.Tide.Chart)) // YYYY-MM-DD なので文字列の順が日付の順

	days := make([]entity.DailyTide, 0, len(dates))
	for _, d := range dates {
		r := raw.Tide.Chart[d]
		days = append(days, entity.DailyTide{
			Date: d,
			Sun: entity.SunInfo{
				AstroTwilight:   r.Sun.AstroTwilight,
				RegularTwilight: r.Sun.RegularTwilight,
				Rise:            r.Sun.Rise,
				Midline:         r.Sun.Midline,
				Set:             r.Sun.Set,
			},
			Moon: entity.MoonInfo{
				Brightness: r.Moon.Brightness,
				Age:        r.Moon.Age,
				Title:      r.Moon.Title,
				Rise:       r.Moon.Rise,
				Midline:    r.Moon.Midline,
				Set:        r.Moon.Set,
				Name:       r.Moon.Name,
			},
			Tide:  toTideEvents(r.Tide),
			Flood: toTideEvents(r.Flood),
			Ebb:   toTideEvents(r.Edd),
		})
	}

	return &entity.TideData{
		Port: entity.TidePort{
			PrefectureCode: raw.Tide.Port.PrefectureCode.String(),
			PortCode:       raw.Tide.Port.HarborCode.String(),
			PortName:       raw.Tide.Port.HarborNameJ,
		},
		Days:     days,
		DataTime: time.Now(),
	}
}

func toTideEvents(in []tide736Event) []entity.TideEvent {
	out := make([]entity.TideEvent, len(in))
	for i, e := range in {
		out[i] = entity.TideEvent{Time: e.Time, UnixMs: e.Unix, CM: e.CM}
	}
	return out
}
