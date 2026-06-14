package external

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
}

func newRetryTransport(maxRetries int) http.RoundTripper {
	return &retryTransport{
		base:       http.DefaultTransport,
		maxRetries: maxRetries,
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	for i := 0; i <= t.maxRetries; i++ {
		if err := req.Context().Err(); err != nil {
			return nil, fmt.Errorf("request context canceled: %w", err)
		}
		if i > 0 {
			select {
			case <-time.After(time.Duration(i) * 500 * time.Millisecond):
			case <-req.Context().Done():
				return nil, fmt.Errorf("request context canceled during backoff: %w", req.Context().Err())
			}
		}
		resp, err := t.base.RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("server error: status %d", resp.StatusCode)
		}
		if resp != nil {
			// リトライループ内: 失敗確定レスポンスを読み捨てて close する。
			// ここで close エラーを slog.Warn 化すると最大 maxRetries 回 Warn が
			// 連続出力され、本物のリトライ失敗ログを埋もれさせる懸念があるため
			// nolint で抑止する（io.Copy も同方針）。
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
			resp.Body.Close()              //nolint:errcheck
		}
	}
	return nil, lastErr
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: newRetryTransport(3),
	}
}
