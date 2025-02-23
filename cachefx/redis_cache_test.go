package cachefx_test

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/eser/ajan/cachefx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*cachefx.RedisCache, *miniredis.Miniredis, error) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		return nil, nil, err //nolint:wrapcheck
	}

	cache, err := cachefx.NewRedisCache(t.Context(), cachefx.DialectRedis, "redis://"+mr.Addr())
	if err != nil {
		mr.Close()

		return nil, nil, err //nolint:wrapcheck
	}

	return cache, mr, nil
}

func TestNewRedisCache(t *testing.T) {
	t.Parallel()

	cache, mr, err := setupTestRedis(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	assert.NotNil(t, cache)
	assert.Equal(t, cachefx.DialectRedis, cache.GetDialect())
}

func TestRedisCache_Set(t *testing.T) {
	t.Parallel()

	cache, mr, err := setupTestRedis(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	err = cache.Set(t.Context(), "test-key", "test-value", time.Minute)
	require.NoError(t, err)

	// Verify the value was set in miniredis
	val, err := mr.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

func TestRedisCache_Get(t *testing.T) {
	t.Parallel()

	cache, mr, err := setupTestRedis(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	// Test getting non-existent key
	val, err := cache.Get(t.Context(), "non-existent")
	require.NoError(t, err)
	assert.Empty(t, val)

	// Set and get a value
	err = mr.Set("test-key", "test-value")
	require.NoError(t, err)

	val, err = cache.Get(t.Context(), "test-key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

func TestRedisCache_Delete(t *testing.T) {
	t.Parallel()

	cache, mr, err := setupTestRedis(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	// Set a value first
	err = mr.Set("test-key", "test-value")
	require.NoError(t, err)

	// Delete the key
	err = cache.Delete(t.Context(), "test-key")
	require.NoError(t, err)

	// Verify the key was deleted
	exists := mr.Exists("test-key")
	assert.False(t, exists)
}
