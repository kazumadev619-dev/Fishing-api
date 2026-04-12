package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedis(t *testing.T) *CacheClient {
	t.Helper()
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	redisURL := fmt.Sprintf("redis://%s:%s", host, port.Port())
	client, err := NewCacheClient(ctx, redisURL)
	require.NoError(t, err)
	return client
}

func TestCacheClient_SetAndGet(t *testing.T) {
	client := setupRedis(t)

	ctx := context.Background()
	key := "test:cache:set-get"
	value := []byte(`{"test": "value"}`)

	err := client.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	result, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestCacheClient_Get_Miss(t *testing.T) {
	client := setupRedis(t)

	ctx := context.Background()
	result, err := client.Get(ctx, "test:cache:nonexistent-key")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCacheClient_Delete(t *testing.T) {
	client := setupRedis(t)

	ctx := context.Background()
	key := "test:cache:delete"

	err := client.Set(ctx, key, []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	err = client.Delete(ctx, key)
	require.NoError(t, err)

	result, err := client.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, result)
}
