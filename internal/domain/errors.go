package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidToken     = errors.New("invalid or expired token")
	ErrEmailNotVerified = errors.New("email not verified")
	// ErrDBUnavailable は DB に繋がらない・応答しないときのエラー。ハンドラは 503 にする。
	ErrDBUnavailable = errors.New("database unavailable")
)

// UpstreamErrorKind は外部 API の失敗の種別。値は Next の ApiErrorType と同じ文字列で、応答の code に出る。
type UpstreamErrorKind string

const (
	UpstreamTimeout      UpstreamErrorKind = "TIMEOUT"
	UpstreamRateLimited  UpstreamErrorKind = "RATE_LIMITED"
	UpstreamServerError  UpstreamErrorKind = "SERVER_ERROR"
	UpstreamClientError  UpstreamErrorKind = "CLIENT_ERROR"
	UpstreamNetworkError UpstreamErrorKind = "NETWORK_ERROR"
)

// UpstreamError は外部 API の失敗。ハンドラはタイムアウトを 504、それ以外を 502 にする。
// Err には上流の本文などが入りうるので、応答には出さずログにだけ残す。
type UpstreamError struct {
	Kind       UpstreamErrorKind
	StatusCode int // 上流の HTTP ステータス。無いときは 0
	Retryable  bool
	Err        error
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream %s (status %d): %v", e.Kind, e.StatusCode, e.Err)
}

func (e *UpstreamError) Unwrap() error { return e.Err }

// NewInvalidUpstreamResponse は、上流が 200 を返したが中身が使えないときのエラー（Next の createApiError(502, …) と同じ扱い）。
func NewInvalidUpstreamResponse(format string, args ...any) *UpstreamError {
	return &UpstreamError{Kind: UpstreamServerError, StatusCode: 502, Retryable: true, Err: fmt.Errorf(format, args...)}
}
