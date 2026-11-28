package handler

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/openapi"
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

func tideBadRequest(msg string) openapi.GetTide400JSONResponse {
	status := http.StatusBadRequest
	return openapi.GetTide400JSONResponse{
		ErrorJSONResponse: openapi.ErrorJSONResponse{
			Error:  msg,
			Status: &status,
		},
	}
}

// tideQuery はクエリを受け取り、既定値（range は week、date は JST の今日）を入れて確かめる。
// 不正なら Next と同じ文言を返す。
func tideQuery(prefCode, portCode, date, tideRange string) (string, string, string, string, string) {
	if date == "" {
		date = time.Now().In(jst).Format("2006-01-02")
	}
	if tideRange == "" {
		tideRange = "week"
	}
	var errMsg string
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

func (h *TideHandler) GetTide(ctx context.Context, req openapi.GetTideRequestObject) (openapi.GetTideResponseObject, error) {
	reqCtx, path := requestContext(ctx)

	var dateParam, rangeParam string
	if req.Params.Date != nil {
		dateParam = *req.Params.Date
	}
	if req.Params.Range != nil {
		rangeParam = string(*req.Params.Range)
	}

	prefCode, portCode, date, tideRange, errMsg := tideQuery(req.Params.PrefectureCode, req.Params.PortCode, dateParam, rangeParam)
	if errMsg != "" {
		return tideBadRequest(errMsg), nil
	}

	data, fromCache, err := h.usecase.GetTideData(reqCtx, prefCode, portCode, date, tideRange)
	if err != nil {
		if status, resp, ok := classifyError(reqCtx, path, err); ok {
			return tideStatusFromClassified(status, resp), nil
		}
		slog.ErrorContext(reqCtx, "Tide API error", "error", err)
		status := http.StatusInternalServerError
		return openapi.GetTide500JSONResponse{
			Error:  "Failed to fetch tide data",
			Status: &status,
		}, nil
	}

	// private にする。認証が要るルートなので、共有キャッシュに保存させない（Next と同じ）。
	cacheControl := "private, max-age=300"
	if fromCache {
		cacheControl = "private, max-age=3600"
	}
	return openapi.GetTide200JSONResponse{
		Body: openapi.TideResponse{
			Status: openapi.N200,
			Data:   toTideData(data),
			Meta:   openapi.CacheMeta{FromCache: fromCache, FetchedAt: isoTime(time.Now())},
		},
		Headers: openapi.GetTide200ResponseHeaders{
			CacheControl: cacheControl,
		},
	}, nil
}

// isValidDate は YYYY-MM-DD で、実在する日かを確かめる（time.Parse は 2 月 30 日を拒否する）。
func isValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func toTideData(d *entity.TideData) openapi.TideData {
	tides := make([]openapi.TideDay, len(d.Days))
	for i, day := range d.Days {
		tides[i] = openapi.TideDay{
			Date: day.Date,
			Daily: openapi.TideDaily{
				Sun: openapi.TideSun{
					AstroTwilight:   day.Sun.AstroTwilight,
					RegularTwilight: day.Sun.RegularTwilight,
					Rise:            day.Sun.Rise,
					Midline:         day.Sun.Midline,
					Set:             day.Sun.Set,
				},
				Moon: openapi.TideMoon{
					Brightness: day.Moon.Brightness,
					Age:        day.Moon.Age,
					Title:      day.Moon.Title,
					Rise:       day.Moon.Rise,
					Midline:    day.Moon.Midline,
					Set:        day.Moon.Set,
					Name:       day.Moon.Name,
				},
				Tide:  toTideEvents(day.Tide),
				Flood: toTideEvents(day.Flood),
				Edd:   toTideEvents(day.Ebb),
			},
		}
	}
	return openapi.TideData{
		Port: openapi.TidePort{
			PrefectureCode: d.Port.PrefectureCode,
			PortCode:       d.Port.PortCode,
			PortName:       d.Port.PortName,
		},
		Tides:    tides,
		DataTime: isoTime(d.DataTime),
	}
}

func toTideEvents(in []entity.TideEvent) []openapi.TideEvent {
	out := make([]openapi.TideEvent, len(in))
	for i, e := range in {
		out[i] = openapi.TideEvent{Time: e.Time, Unix: e.UnixMs, Cm: e.CM}
	}
	return out
}
