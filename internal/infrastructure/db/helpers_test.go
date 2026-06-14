package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB は testcontainers で PostgreSQL を起動し、schema.sql を適用した
// *sql.DB と後始末用のクリーンアップ関数を返すのだ。
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:17",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := NewPool(ctx, connStr)
	require.NoError(t, err)

	db := stdlib.OpenDBFromPool(pool)

	// Apply schema
	schema, err := os.ReadFile("../../../db/schema.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(schema))
	require.NoError(t, err)

	return db, func() {
		db.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate test container: %v", err)
		}
	}
}

// seedLocation は locations テーブルに直接 1 件挿入し、その ID を返すのだ。
// locations は repo 経由で作らず、テストのセットアップとして DB に直接 INSERT するのだ。
func seedLocation(t *testing.T, db *sql.DB, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	id := uuid.New()
	_, err := db.ExecContext(ctx,
		`INSERT INTO locations (id, name, latitude, longitude, location_type)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, name, 35.0, 139.0, "SHORE",
	)
	require.NoError(t, err)
	return id
}

// seedPort は ports テーブルに直接 1 件挿入し、その ID を返すのだ。
func seedPort(t *testing.T, db *sql.DB, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	id := uuid.New()
	_, err := db.ExecContext(ctx,
		`INSERT INTO ports (id, name, prefecture_code, port_code)
		 VALUES ($1, $2, $3, $4)`,
		id, name, "13", "P001",
	)
	require.NoError(t, err)
	return id
}

// seedFullLocation は region・prefecture・port_id まで埋めた location を挿入するのだ。
// toLocationEntity の nil でない分岐をカバーするために使うのだ。
func seedFullLocation(t *testing.T, db *sql.DB, name string, portID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	id := uuid.New()
	_, err := db.ExecContext(ctx,
		`INSERT INTO locations (id, name, latitude, longitude, region, prefecture, location_type, port_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, name, 35.0, 139.0, "Kanto", "Tokyo", "PORT", portID,
	)
	require.NoError(t, err)
	return id
}
