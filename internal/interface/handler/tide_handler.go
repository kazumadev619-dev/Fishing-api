package handler

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type TideFetcher interface {
	GetTideData(ctx context.Context, prefCode, portCode, date, tideRange string) (*entity.TideData, bool, error)
}

type TideHandler struct {
	usecase TideFetcher
}

func NewTideHandler(uc TideFetcher) *TideHandler {
	return &TideHandler{usecase: uc}
}

var (
	prefCodeRegex = regexp.MustCompile(`^[0-9]{1,2}$`)
	portCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	tideRanges    = []string{"day", "week", "month"}
	// 日本は夏時間が無いので固定のずれでよい。
	jst = time.FixedZone("JST", 9*60*60)
)

// 応答の形は Next の /api/conditions/tide（src/lib/tideService.ts の FormattedTideData）と同じ。
type tideResponse struct {
	Status int       `json:"status"`
	Data   tideBody  `json:"data"`
	Meta   cacheMeta `json:"meta"`
}

type cacheMeta struct {
	FromCache bool   `json:"fromCache"`
	FetchedAt string `json:"fetchedAt"`
}

type tideBody struct {
	Port struct {
		PrefectureCode string `json:"prefecture_code"`
		PortCode       string `json:"port_code"`
		PortName       string `json:"port_name"`
	} `json:"port"`
	Tides    []tideDay `json:"tides"`
	DataTime string    `json:"dataTime"`
}

type tideDay struct {
	Date  string `json:"date"`
	Daily struct {
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
		Tide  []tideEvent `json:"tide"`
		Flood []tideEvent `json:"flood"`
		Edd   []tideEvent `json:"edd"`
	} `json:"daily"`
}

type tideEvent struct {
	Time string  `json:"time"`
	Unix int64   `json:"unix"`
	CM   float64 `json:"cm"`
}

// tideQuery はクエリを読み、既定値（range は week、date は JST の今日）を入れて確かめる。
// 不正なら Next と同じ文言を返す。
func tideQuery(c *gin.Context) (prefCode, portCode, date, tideRange, errMsg string) {
	prefCode, portCode = c.Query("prefectureCode"), c.Query("portCode")
	date, tideRange = c.Query("date"), c.Query("range")
	if date == "" {
		date = time.Now().In(jst).Format("2006-01-02")
	}
	if tideRange == "" {
		tideRange = "week"
	}
	switch {
	case prefCode == "":
		errMsg = "Missing required parameter: prefectureCode"
	case portCode == "":
		errMsg = "Missing required parameter: portCode"
	case !prefCodeRegex.MatchString(prefCode):
		errMsg = "Invalid prefectureCode format"
	case !portCodeRegex.MatchString(portCode):
		errMsg = "Invalid portCode format"
	case !isValidDate(date):
		errMsg = "Invalid date. Use an existing date in YYYY-MM-DD format"
	case !slices.Contains(tideRanges, tideRange):
		errMsg = "Invalid range. Must be one of: day, week, month"
	}
	return prefCode, portCode, date, tideRange, errMsg
}

func (h *TideHandler) Get(c *gin.Context) {
	prefCode, portCode, date, tideRange, errMsg := tideQuery(c)
	if errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg, "status": http.StatusBadRequest})
		return
	}

	data, fromCache, err := h.usecase.GetTideData(c.Request.Context(), prefCode, portCode, date, tideRange)
	if err != nil {
		if respondClassifiedError(c, err, nil) {
			return
		}
		slog.ErrorContext(c.Request.Context(), "Tide API error", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tide data", "status": http.StatusInternalServerError})
		return
	}

	// private にする。認証が要るルートなので、共有キャッシュに保存させない（Next と同じ）。
	if fromCache {
		c.Header("Cache-Control", "private, max-age=3600")
	} else {
		c.Header("Cache-Control", "private, max-age=300")
	}
	c.JSON(http.StatusOK, tideResponse{
		Status: http.StatusOK,
		Data:   toTideBody(data),
		Meta:   cacheMeta{FromCache: fromCache, FetchedAt: isoTime(time.Now())},
	})
}

// isValidDate は YYYY-MM-DD で、実在する日かを確かめる（time.Parse は 2 月 30 日を拒否する）。
func isValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func toTideBody(d *entity.TideData) tideBody {
	var b tideBody
	b.Port.PrefectureCode = d.Port.PrefectureCode
	b.Port.PortCode = d.Port.PortCode
	b.Port.PortName = d.Port.PortName
	b.DataTime = isoTime(d.DataTime)
	b.Tides = make([]tideDay, len(d.Days))
	for i, day := range d.Days {
		t := &b.Tides[i]
		t.Date = day.Date
		t.Daily.Sun.AstroTwilight = day.Sun.AstroTwilight
		t.Daily.Sun.RegularTwilight = day.Sun.RegularTwilight
		t.Daily.Sun.Rise = day.Sun.Rise
		t.Daily.Sun.Midline = day.Sun.Midline
		t.Daily.Sun.Set = day.Sun.Set
		t.Daily.Moon.Brightness = day.Moon.Brightness
		t.Daily.Moon.Age = day.Moon.Age
		t.Daily.Moon.Title = day.Moon.Title
		t.Daily.Moon.Rise = day.Moon.Rise
		t.Daily.Moon.Midline = day.Moon.Midline
		t.Daily.Moon.Set = day.Moon.Set
		t.Daily.Moon.Name = day.Moon.Name
		t.Daily.Tide = toTideEvents(day.Tide)
		t.Daily.Flood = toTideEvents(day.Flood)
		t.Daily.Edd = toTideEvents(day.Ebb)
	}
	return b
}

func toTideEvents(in []entity.TideEvent) []tideEvent {
	out := make([]tideEvent, len(in))
	for i, e := range in {
		out[i] = tideEvent{Time: e.Time, Unix: e.UnixMs, CM: e.CM}
	}
	return out
}
