package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	_ "time/tzdata"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

// TideClient は tide736.net の潮位 API クライアントのだ。
type TideClient struct {
	baseURL string
	client  *http.Client
}

// NewTideClient は本番用クライアントを返すのだ。
func NewTideClient() *TideClient {
	return &TideClient{
		baseURL: "https://tide736.net",
		client:  newHTTPClient(),
	}
}

func newTideClientWithHTTPClient(baseURL string, httpClient *http.Client) *TideClient {
	return &TideClient{baseURL: baseURL, client: httpClient}
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

// FetchTideData は指定した都道府県コード・港コード・日付の潮位データを取得するのだ。
func (c *TideClient) FetchTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error) {
	url := fmt.Sprintf("%s/api/get_tide/%s/%s/%s/", c.baseURL, prefCode, portCode, date)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating tide request: %w", err)
	}
	resp, err := c.client.Do(req)
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

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return nil, fmt.Errorf("loading Asia/Tokyo location: %w", err)
	}
	dateBase, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return nil, fmt.Errorf("parsing date %q: %w", date, err)
	}

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
