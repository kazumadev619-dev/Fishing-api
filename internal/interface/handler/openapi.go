package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/openapi"
	"github.com/kazumadev619-dev/fishing-api/pkg/validator"
)

type openAPIServer struct {
	*WeatherHandler
	*TideHandler
}

var _ openapi.StrictServerInterface = (*openAPIServer)(nil)

// RegisterOpenAPI は api/openapi.yaml に定義したルートを Gin ルーターへ登録する。
// 生成されたラッパーは必須クエリの欠落や型変換失敗時にハンドラを呼ばず ErrorHandler へ落とすため、
// ルートごとのフォールバックで段2（Next 互換）と同じ評価順・エラー文言を返す（ADR 0008）。
func RegisterOpenAPI(r gin.IRouter, w *WeatherHandler, t *TideHandler) {
	srv := openapi.NewStrictHandler(&openAPIServer{
		WeatherHandler: w,
		TideHandler:    t,
	}, nil)

	openapi.RegisterHandlersWithOptions(r, srv, openapi.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			ctx := c.Request.Context()
			switch c.FullPath() {
			case "/api/weather":
				if w != nil && !w.apiConfigured {
					writeOpenAPIResponse(ctx, weatherNotConfigured(ctx).VisitGetWeatherResponse(c.Writer))
					return
				}
				_, _, parseErr := validator.ParseAndValidateCoordinates(c.Query("lat"), c.Query("lon"))
				msg := err.Error()
				if parseErr != nil {
					msg = parseErr.Error()
				}
				writeOpenAPIResponse(ctx, weatherBadRequest(msg).VisitGetWeatherResponse(c.Writer))
			case "/api/conditions/tide":
				_, _, _, _, errMsg := tideQuery(
					c.Query("prefectureCode"),
					c.Query("portCode"),
					c.Query("date"),
					c.Query("range"),
				)
				if errMsg == "" {
					errMsg = err.Error()
				}
				writeOpenAPIResponse(ctx, tideBadRequest(errMsg).VisitGetTideResponse(c.Writer))
			default:
				c.JSON(statusCode, openapi.ErrorResponse{Error: err.Error()})
			}
		},
	})
}

func writeOpenAPIResponse(ctx context.Context, err error) {
	if err != nil {
		slog.ErrorContext(ctx, "failed to write OpenAPI error response", "error", err)
	}
}

// requestContext は strict server が渡す ctx（実体は *gin.Context）から
// リクエストコンテキストと登録済みパスを取り出す。
func requestContext(ctx context.Context) (context.Context, string) {
	if c, ok := ctx.(*gin.Context); ok && c.Request != nil {
		return c.Request.Context(), c.FullPath()
	}
	return ctx, ""
}

func weatherStatusFromClassified(status int, resp openapi.ErrorResponse) openapi.GetWeatherResponseObject {
	switch status {
	case http.StatusServiceUnavailable:
		return openapi.GetWeather503JSONResponse(resp)
	case http.StatusGatewayTimeout:
		f := false
		resp.Success = &f
		return openapi.GetWeather504JSONResponse(resp)
	default:
		f := false
		resp.Success = &f
		return openapi.GetWeather502JSONResponse(resp)
	}
}

func tideStatusFromClassified(status int, resp openapi.ErrorResponse) openapi.GetTideResponseObject {
	switch status {
	case http.StatusServiceUnavailable:
		return openapi.GetTide503JSONResponse(resp)
	case http.StatusGatewayTimeout:
		return openapi.GetTide504JSONResponse(resp)
	default:
		return openapi.GetTide502JSONResponse(resp)
	}
}
