package handler

import (
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
)

// 利用者向けの固定文言。Next の getErrorMessage（src/lib/apiErrorUtils.ts）と同じ。
var upstreamMessages = map[domain.UpstreamErrorKind]string{
	domain.UpstreamTimeout:      "通信がタイムアウトしました。インターネット接続を確認して、もう一度お試しください。",
	domain.UpstreamRateLimited:  "API の使用制限に達しました。数分後にお試しください。",
	domain.UpstreamServerError:  "サーバーで問題が発生しました。サポートに連絡してください。",
	domain.UpstreamClientError:  "リクエストが無効です。入力内容を確認してください。",
	domain.UpstreamNetworkError: "ネットワーク接続を確認してください。",
}

// respondClassifiedError は「異常なエラー」を応答にする。上流の失敗は 502、上流のタイムアウトは 504、
// DB の接続の失敗は 503（Next の docs/reference/api-error-handling.md）。
// extra は応答に足す項目（weather の success: false など）。分類できないエラーは何も書かずに false を返す。
func respondClassifiedError(c *gin.Context, err error, extra gin.H) bool {
	var ue *domain.UpstreamError
	switch {
	case errors.As(err, &ue):
		slog.ErrorContext(c.Request.Context(), "API request failed",
			"path", c.FullPath(), "errorType", ue.Kind, "statusCode", ue.StatusCode, "retryable", ue.Retryable, "error", err)
		status := http.StatusBadGateway
		if ue.Kind == domain.UpstreamTimeout {
			status = http.StatusGatewayTimeout
		}
		body := gin.H{}
		maps.Copy(body, extra)
		body["error"] = upstreamMessages[ue.Kind]
		body["code"] = ue.Kind
		body["retryable"] = ue.Retryable
		c.JSON(status, body)
		return true
	case errors.Is(err, domain.ErrDBUnavailable):
		slog.ErrorContext(c.Request.Context(), "Database request failed", "path", c.FullPath(), "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "一時的に利用できません。時間をおいて再度お試しください。",
			"code":  "DB_UNAVAILABLE",
		})
		return true
	}
	return false
}

// isoTime は JavaScript の Date#toJSON と同じ形（UTC、ミリ秒まで）にする。
func isoTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
}
