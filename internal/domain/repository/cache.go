package repository

import (
	"context"
	"time"
)

// Cache はキャッシュ操作のインターフェース。
// usecase 層が依存する。infrastructure/cache.CacheClient がこれを実装する。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
