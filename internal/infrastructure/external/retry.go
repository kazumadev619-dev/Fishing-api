package external

import (
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
		if i > 0 {
			time.Sleep(time.Duration(i) * 500 * time.Millisecond)
		}
		resp, err := t.base.RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = nil
		}
		if resp != nil {
			resp.Body.Close()
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
