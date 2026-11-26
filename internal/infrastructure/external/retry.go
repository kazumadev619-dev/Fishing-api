package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/kazumadev619-dev/fishing-api/internal/domain"
)

// Next の src/lib/api/client.ts と同じ決まり：1回 10 秒、リトライ 3 回（計 4 回）、待ちは 100・200・400ms。
const (
	maxRetries     = 3
	attemptTimeout = 10 * time.Second
	baseBackoff    = 100 * time.Millisecond
)

// 429 は入れない。絞られている上流を叩き直すと呼び出しが増えるだけ（Next #317）。
var retryableStatuses = []int{
	http.StatusRequestTimeout,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}

// fetcher は外部 API に GET して JSON を読む。失敗は *domain.UpstreamError に分類し、対象ならリトライする。
type fetcher struct {
	client         *http.Client
	attemptTimeout time.Duration
	backoff        time.Duration
}

func newFetcher(c *http.Client) *fetcher {
	return &fetcher{client: c, attemptTimeout: attemptTimeout, backoff: baseBackoff}
}

// getJSON は rawURL を GET し、本文を out に読む。
// rawURL には API キーが入りうるので、エラーにもログにも URL を含めない。
func (f *fetcher) getJSON(ctx context.Context, rawURL string, out any) error {
	var lastErr *domain.UpstreamError
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := f.backoff << (attempt - 1)
			slog.Warn("retrying upstream request", "attempt", attempt, "maxRetries", maxRetries, "delay", delay, "kind", lastErr.Kind, "status", lastErr.StatusCode)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("upstream request canceled during backoff: %w", ctx.Err())
			}
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("upstream request canceled: %w", err)
		}

		body, err := f.attempt(ctx, rawURL)
		if err == nil {
			if jerr := json.Unmarshal(body, out); jerr != nil {
				return domain.NewInvalidUpstreamResponse("decoding upstream response: %w", jerr)
			}
			return nil
		}
		if !errors.As(err, &lastErr) {
			return err // 呼び出し元の取り消しなど、上流の失敗ではないもの
		}
		if !lastErr.Retryable {
			return lastErr
		}
	}
	return lastErr
}

// attempt は1回分の GET。1回ごとに期限を掛ける（全体に掛けると、遅い1回で残りのリトライが無くなる）。
func (f *fetcher) attempt(ctx context.Context, rawURL string) ([]byte, error) {
	actx, cancel := context.WithTimeout(ctx, f.attemptTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(actx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating upstream request: %w", stripURL(err))
	}
	resp, err := f.client.Do(req)
	if err == nil {
		defer resp.Body.Close() //nolint:errcheck // 読み終えた本文の close 失敗は結果に影響しない
		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err == nil {
			return body, classifyStatus(resp.StatusCode)
		}
	}
	switch {
	case ctx.Err() != nil:
		return nil, fmt.Errorf("upstream request canceled: %w", ctx.Err())
	case errors.Is(actx.Err(), context.DeadlineExceeded):
		return nil, &domain.UpstreamError{
			Kind: domain.UpstreamTimeout, Retryable: true,
			Err: fmt.Errorf("request timeout after %s", f.attemptTimeout),
		}
	default:
		return nil, &domain.UpstreamError{Kind: domain.UpstreamNetworkError, Retryable: true, Err: stripURL(err)}
	}
}

// classifyStatus は Next の createApiError と同じ分け方。200 番台は nil。
func classifyStatus(status int) error {
	if status >= 200 && status < 300 {
		return nil
	}
	e := &domain.UpstreamError{
		StatusCode: status,
		Retryable:  slices.Contains(retryableStatuses, status),
		Err:        fmt.Errorf("upstream returned status %d", status),
	}
	switch {
	case status == http.StatusTooManyRequests:
		e.Kind = domain.UpstreamRateLimited
	case status < 500:
		e.Kind = domain.UpstreamClientError
	default:
		e.Kind = domain.UpstreamServerError
	}
	return e
}

// stripURL は *url.Error から URL（API キー入り）を外し、中身のエラーだけを返す。
func stripURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("%s: %w", ue.Op, ue.Err)
	}
	return err
}
