package external

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryTransport_SuccessOnFirstTry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newHTTPClient()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, calls)
}

// retryTransport は同一の *http.Request を maxRetries+1 回 RoundTrip に渡す。
// 本ファイル内のテストは全て GET + Body=nil なので req 再利用は安全。
// 将来 POST/PUT などボディ付きリクエストのリトライテストを追加する場合は、
// http.Request.Body が単発の io.ReadCloser である関係で 2 回目以降の RoundTrip で
// EOF となりリトライが失敗する。その際は retry.go 側で req.Clone(ctx) する実装、
// もしくはテスト内で req.GetBody を設定する等の対応が別途必要になる。
func TestRetryTransport_RetryOn5xx(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newHTTPClient()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	// retryTransport は全リトライ失敗時に必ず (nil, err) を返すため resp は nil 想定。
	// 将来 retry.go の戻り値仕様が変わっても body リークを起こさないよう防御的に close。
	if resp != nil {
		defer resp.Body.Close()
	}
	// 4回試行（0+3retry）してすべて5xxならエラーを返す
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, 4, calls)
}

func TestRetryTransport_SuccessOnRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newHTTPClient()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, calls)
}
