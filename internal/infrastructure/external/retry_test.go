package external

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 上流のステータスごとに、エラーの種別・リトライの有無・呼ばれた回数を確かめる。
// リトライの対象は 408・500・502・503・504（Next の RETRYABLE_STATUS_CODES）。429・501・505 は対象外。
func TestGetJSON_StatusClassification(t *testing.T) {
	tests := []struct {
		status        int
		wantKind      domain.UpstreamErrorKind
		wantRetryable bool
		wantCalls     int32
	}{
		{http.StatusRequestTimeout, domain.UpstreamClientError, true, 4},
		{http.StatusInternalServerError, domain.UpstreamServerError, true, 4},
		{http.StatusBadGateway, domain.UpstreamServerError, true, 4},
		{http.StatusServiceUnavailable, domain.UpstreamServerError, true, 4},
		{http.StatusGatewayTimeout, domain.UpstreamServerError, true, 4},
		{http.StatusTooManyRequests, domain.UpstreamRateLimited, false, 1},
		{http.StatusNotImplemented, domain.UpstreamServerError, false, 1},
		{http.StatusHTTPVersionNotSupported, domain.UpstreamServerError, false, 1},
		{http.StatusNotFound, domain.UpstreamClientError, false, 1},
		{http.StatusUnauthorized, domain.UpstreamClientError, false, 1},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			var out map[string]any
			err := fastFetcher(srv.Client()).getJSON(t.Context(), srv.URL, &out)

			var ue *domain.UpstreamError
			require.ErrorAs(t, err, &ue)
			assert.Equal(t, tt.wantKind, ue.Kind)
			assert.Equal(t, tt.status, ue.StatusCode)
			assert.Equal(t, tt.wantRetryable, ue.Retryable)
			assert.Equal(t, tt.wantCalls, calls.Load())
		})
	}
}

func TestGetJSON_SucceedsAfterRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var out struct{ OK bool }
	require.NoError(t, fastFetcher(srv.Client()).getJSON(t.Context(), srv.URL, &out))
	assert.True(t, out.OK)
	assert.Equal(t, int32(3), calls.Load())
}

// 1回ごとの期限を過ぎたらタイムアウトとして扱い、リトライする。全部だめなら TIMEOUT。
func TestGetJSON_PerAttemptTimeout(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-r.Context().Done()
	}))
	defer srv.Close()

	f := fastFetcher(srv.Client())
	f.attemptTimeout = 20 * time.Millisecond
	err := f.getJSON(t.Context(), srv.URL, &struct{}{})

	var ue *domain.UpstreamError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, domain.UpstreamTimeout, ue.Kind)
	assert.True(t, ue.Retryable)
	assert.Equal(t, int32(4), calls.Load())
}

// 1回ごとの期限で、最初の試みが遅くても2回目で取れる（全体に1つの期限を掛けていた頃はここで失敗した）。
func TestGetJSON_SlowFirstAttemptThenSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			<-r.Context().Done()
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	f := fastFetcher(srv.Client())
	f.attemptTimeout = 50 * time.Millisecond
	require.NoError(t, f.getJSON(t.Context(), srv.URL, &struct{}{}))
	assert.Equal(t, int32(2), calls.Load())
}

func TestGetJSON_NetworkErrorIsRetried(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // 繋がらない宛先にする

	err := fastFetcher(http.DefaultClient).getJSON(t.Context(), url, &struct{}{})

	var ue *domain.UpstreamError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, domain.UpstreamNetworkError, ue.Kind)
	assert.True(t, ue.Retryable)
}

// 呼び出し元が取り消したら、リトライせずに context のエラーを返す（上流の失敗ではない）。
func TestGetJSON_CallerCanceled(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := fastFetcher(srv.Client()).getJSON(ctx, srv.URL, &struct{}{})

	assert.ErrorIs(t, err, context.Canceled)
	var ue *domain.UpstreamError
	assert.False(t, errors.As(err, &ue))
	assert.Equal(t, int32(0), calls.Load())
}

func TestGetJSON_InvalidJSONIsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>`))
	}))
	defer srv.Close()

	err := fastFetcher(srv.Client()).getJSON(t.Context(), srv.URL, &struct{}{})

	var ue *domain.UpstreamError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, domain.UpstreamServerError, ue.Kind)
}

// fastFetcher はテスト用に待ち時間を縮めた fetcher を返す。
func fastFetcher(c *http.Client) *fetcher {
	return &fetcher{client: c, attemptTimeout: 2 * time.Second, backoff: time.Millisecond}
}

// 2xx 以外の本文の先頭は、原因を追えるよう UpstreamError.Err に残す（ログ用。応答には出ない）。長い本文は切る。
func TestGetJSON_KeepsErrorBodyPrefix(t *testing.T) {
	long := strings.Repeat("x", 2000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"cod":401,"message":"Invalid API key."}` + long))
	}))
	defer srv.Close()

	err := fastFetcher(srv.Client()).getJSON(t.Context(), srv.URL, &struct{}{})

	var ue *domain.UpstreamError
	require.ErrorAs(t, err, &ue)
	assert.Contains(t, ue.Err.Error(), `"message":"Invalid API key."`)
	assert.Less(t, len(ue.Err.Error()), 700)
}
