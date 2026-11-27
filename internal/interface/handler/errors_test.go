package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// エラーの3分類：上流の失敗は 502、上流のタイムアウトは 504、DB の接続の失敗は 503。
// どれでもなければ false を返し、呼び出し側が 500 にする。
func TestRespondClassifiedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := func(kind domain.UpstreamErrorKind, status int, retryable bool) error {
		// ハンドラにはユースケースで包まれて届く
		return fmt.Errorf("fetching: %w", &domain.UpstreamError{Kind: kind, StatusCode: status, Retryable: retryable, Err: errors.New("upstream body secret")})
	}
	tests := []struct {
		name       string
		err        error
		wantOK     bool
		wantStatus int
		wantBody   string
	}{
		{
			"タイムアウトは 504", upstream(domain.UpstreamTimeout, 0, true), true, 504,
			`{"error":"通信がタイムアウトしました。インターネット接続を確認して、もう一度お試しください。","code":"TIMEOUT","retryable":true}`,
		},
		{
			"上流の 5xx は 502", upstream(domain.UpstreamServerError, 503, true), true, 502,
			`{"error":"サーバーで問題が発生しました。サポートに連絡してください。","code":"SERVER_ERROR","retryable":true}`,
		},
		{
			"上流の 4xx も 502", upstream(domain.UpstreamClientError, 401, false), true, 502,
			`{"error":"リクエストが無効です。入力内容を確認してください。","code":"CLIENT_ERROR","retryable":false}`,
		},
		{
			"上流の 429 も 502", upstream(domain.UpstreamRateLimited, 429, false), true, 502,
			`{"error":"API の使用制限に達しました。数分後にお試しください。","code":"RATE_LIMITED","retryable":false}`,
		},
		{
			"通信エラーは 502", upstream(domain.UpstreamNetworkError, 0, true), true, 502,
			`{"error":"ネットワーク接続を確認してください。","code":"NETWORK_ERROR","retryable":true}`,
		},
		{
			"DB の接続の失敗は 503", fmt.Errorf("finding: %w", domain.ErrDBUnavailable), true, 503,
			`{"error":"一時的に利用できません。時間をおいて再度お試しください。","code":"DB_UNAVAILABLE"}`,
		},
		{"想定外は扱わない", errors.New("boom"), false, 0, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

			ok := respondClassifiedError(c, tt.err, nil)

			require.Equal(t, tt.wantOK, ok)
			if !ok {
				assert.False(t, c.Writer.Written())
				return
			}
			assert.Equal(t, tt.wantStatus, w.Code)
			assert.JSONEq(t, tt.wantBody, w.Body.String())
			assert.NotContains(t, w.Body.String(), "secret", "上流の本文を応答に出さない")
		})
	}
}

func TestRespondClassifiedError_ExtraFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	respondClassifiedError(c, &domain.UpstreamError{Kind: domain.UpstreamServerError}, gin.H{"success": false})

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["success"])
	assert.Equal(t, "SERVER_ERROR", body["code"])
}
