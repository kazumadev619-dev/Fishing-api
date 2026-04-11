package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: このテストはRedisが localhost:6379 で起動していることが前提。
// docker-compose up -d redis で起動すること。
func TestCacheClient_SetAndGet(t *testing.T) {
	client, err := NewCacheClient("redis://localhost:6379")
	require.NoError(t, err)

	ctx := context.Background()
	key := "test:cache:set-get"
	value := []byte(`{"test": "value"}`)

	err = client.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	result, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)

	t.Cleanup(func() { client.Delete(ctx, key) })
}

func TestCacheClient_Get_Miss(t *testing.T) {
	client, err := NewCacheClient("redis://localhost:6379")
	require.NoError(t, err)

	ctx := context.Background()
	result, err := client.Get(ctx, "test:cache:nonexistent-key")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCacheClient_Delete(t *testing.T) {
	client, err := NewCacheClient("redis://localhost:6379")
	require.NoError(t, err)

	ctx := context.Background()
	key := "test:cache:delete"

	err = client.Set(ctx, key, []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	err = client.Delete(ctx, key)
	require.NoError(t, err)

	result, err := client.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, result)
}
