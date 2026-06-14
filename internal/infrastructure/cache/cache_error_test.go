package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheClient_InvalidURL(t *testing.T) {
	_, err := NewCacheClient(context.Background(), "::not-a-redis-url::")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse redis URL")
}

func TestNewCacheClient_Unreachable(t *testing.T) {
	// 形式は正しいが接続できないアドレス（ローカルの未使用ポート）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := NewCacheClient(ctx, "redis://127.0.0.1:1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connect to redis")
}

// クローズ済みクライアントに対する操作はエラーを返す（エラーラップ経路の検証）
func TestCacheClient_OperationsAfterClose(t *testing.T) {
	client := setupRedis(t)
	require.NoError(t, client.client.Close())

	ctx := context.Background()

	_, getErr := client.Get(ctx, "key")
	assert.Error(t, getErr)
	assert.Contains(t, getErr.Error(), "cache get error")

	setErr := client.Set(ctx, "key", []byte("v"), time.Minute)
	assert.Error(t, setErr)
	assert.Contains(t, setErr.Error(), "cache set error")

	delErr := client.Delete(ctx, "key")
	assert.Error(t, delErr)
	assert.Contains(t, delErr.Error(), "cache delete error")
}
