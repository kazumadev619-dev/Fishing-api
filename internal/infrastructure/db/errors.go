package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
)

// classifyDBError は、DB に繋がらない・応答しないエラーに domain.ErrDBUnavailable を足す。
// それ以外（クエリの誤り・制約違反など）はそのまま返す。
// 分け方は Next の isDatabaseUnavailableError（src/lib/apiErrorUtils.ts）に合わせた。
func classifyDBError(err error) error {
	if isDBUnavailable(err) {
		return fmt.Errorf("%w: %w", domain.ErrDBUnavailable, err)
	}
	return err
}

func isDBUnavailable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 08: 接続の例外、53: 資源の枯渇、57P01〜57P03: 停止・起動中
		return strings.HasPrefix(pgErr.Code, "08") || strings.HasPrefix(pgErr.Code, "53") ||
			slices.Contains([]string{"57P01", "57P02", "57P03"}, pgErr.Code)
	}
	var connErr *pgconn.ConnectError
	var netErr net.Error
	return errors.As(err, &connErr) || errors.As(err, &netErr) ||
		errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, context.DeadlineExceeded)
}
