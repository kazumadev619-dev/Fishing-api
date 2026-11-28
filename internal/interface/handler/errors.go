package handler

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/openapi"
)

// 利用者向けの固定文言。Next の getErrorMessage（src/lib/apiErrorUtils.ts）と同じ。
var upstreamMessages = map[domain.UpstreamErrorKind]string{
	domain.UpstreamTimeout:      "通信がタイムアウトしました。インターネット接続を確認して、もう一度お試しください。",
	domain.UpstreamRateLimited:  "API の使用制限に達しました。数分後にお試しください。",
	domain.UpstreamServerError:  "サーバーで問題が発生しました。サポートに連絡してください。",
	domain.UpstreamClientError:  "リクエストが無効です。入力内容を確認してください。",
	domain.UpstreamNetworkError: "ネットワーク接続を確認してください。",
}

// classifyError は「異常なエラー」を HTTP ステータスと ErrorResponse に分類する。
// 上流の失敗は 502、上流のタイムアウトは 504、DB の接続の失敗は 503（Next の docs/reference/api-error-handling.md）。
// 分類できないエラーは ok=false を返す。
func classifyError(ctx context.Context, path string, err error) (status int, resp openapi.ErrorResponse, ok bool) {
	var ue *domain.UpstreamError
	switch {
	case errors.As(err, &ue):
		slog.ErrorContext(ctx, "API request failed",
			"path", path, "errorType", ue.Kind, "statusCode", ue.StatusCode, "retryable", ue.Retryable, "error", err)
		status = http.StatusBadGateway
		if ue.Kind == domain.UpstreamTimeout {
			status = http.StatusGatewayTimeout
		}
		code := openapi.ErrorCode(ue.Kind)
		retryable := ue.Retryable
		return status, openapi.ErrorResponse{
			Error:     upstreamMessages[ue.Kind],
			Code:      &code,
			Retryable: &retryable,
		}, true
	case errors.Is(err, domain.ErrDBUnavailable):
		slog.ErrorContext(ctx, "Database request failed", "path", path, "error", err)
		code := openapi.DBUNAVAILABLE
		return http.StatusServiceUnavailable, openapi.ErrorResponse{
			Error: "一時的に利用できません。時間をおいて再度お試しください。",
			Code:  &code,
		}, true
	}
	return 0, openapi.ErrorResponse{}, false
}

// respondClassifiedError は「異常なエラー」を応答にする（favorites など OpenAPI 未移行のハンドラ向け）。
// extra は応答に足す項目（weather の success: false など）。分類できないエラーは何も書かずに false を返す。
func respondClassifiedError(c *gin.Context, err error, extra gin.H) bool {
	status, resp, ok := classifyError(c.Request.Context(), c.FullPath(), err)
	if !ok {
		return false
	}
	if len(extra) == 0 || status == http.StatusServiceUnavailable {
		c.JSON(status, resp)
		return true
	}
	body := gin.H{"error": resp.Error}
	maps.Copy(body, extra)
	if resp.Code != nil {
		body["code"] = *resp.Code
	}
	if resp.Retryable != nil {
		body["retryable"] = *resp.Retryable
	}
	c.JSON(status, body)
	return true
}

// isoTime は JavaScript の Date#toJSON と同じ形（UTC、ミリ秒まで）にする。
func isoTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
}
