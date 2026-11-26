package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/stretchr/testify/assert"
)

// DB に繋がらない・応答しないエラーだけを domain.ErrDBUnavailable にする（ハンドラが 503 にする）。
// クエリの誤りや制約違反は想定外（500）のまま。分け方は Next の isDatabaseUnavailableError に合わせた。
func TestClassifyDBError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		unavailable bool
	}{
		{"接続の拒否", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, true},
		{"包まれた通信エラー", fmt.Errorf("query: %w", &net.OpError{Op: "read", Err: errors.New("reset")}), true},
		{"使えない接続", driver.ErrBadConn, true},
		{"閉じた接続", sql.ErrConnDone, true},
		{"期限切れ", context.DeadlineExceeded, true},
		{"SQLSTATE 08006（接続の失敗）", &pgconn.PgError{Code: "08006"}, true},
		{"SQLSTATE 53300（接続が多すぎる）", &pgconn.PgError{Code: "53300"}, true},
		{"SQLSTATE 57P01（管理者による停止）", &pgconn.PgError{Code: "57P01"}, true},
		{"SQLSTATE 57P03（起動中）", &pgconn.PgError{Code: "57P03"}, true},
		{"SQLSTATE 57014（クエリの取り消し）は対象外", &pgconn.PgError{Code: "57014"}, false},
		{"SQLSTATE 42601（構文の誤り）は対象外", &pgconn.PgError{Code: "42601"}, false},
		{"SQLSTATE 23505（一意制約）は対象外", &pgconn.PgError{Code: "23505"}, false},
		{"行が無いのは対象外", sql.ErrNoRows, false},
		{"呼び出し元の取り消しは対象外", context.Canceled, false},
		{"そのほかは対象外", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyDBError(tt.err)

			assert.Equal(t, tt.unavailable, errors.Is(got, domain.ErrDBUnavailable))
			assert.ErrorIs(t, got, tt.err, "元のエラーも辿れる")
		})
	}
}
