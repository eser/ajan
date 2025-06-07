package cachefx_test

import (
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/eser/ajan/cachefx"
	"github.com/eser/ajan/logfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRegistry(t *testing.T) *cachefx.Registry {
	t.Helper()

	logger, _ := logfx.NewLogger(
		os.Stdout,
		&logfx.Config{}, //nolint:exhaustruct
	)

	return cachefx.NewRegistry(logger)
}

func setupTestRedisServer(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	t.Cleanup(func() {
		mr.Close()
	})

	return mr
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)
	assert.NotNil(t, registry)
}

func TestRegistry_GetDefault(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)
	mr := setupTestRedisServer(t)

	// Initially no default cache
	assert.Nil(t, registry.GetDefault())

	// Add a default cache
	err := registry.AddConnection(
		t.Context(),
		cachefx.DefaultCache,
		"redis",
		"redis://"+mr.Addr(),
	)
	require.NoError(t, err) // Should succeed with miniredis

	// Now default should exist
	assert.NotNil(t, registry.GetDefault())
	assert.Equal(t, cachefx.DialectRedis, registry.GetDefault().GetDialect())
}

func TestRegistry_GetNamed(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)
	mr := setupTestRedisServer(t)

	// Test non-existent cache
	assert.Nil(t, registry.GetNamed("non-existent"))

	// Add a named cache
	err := registry.AddConnection(t.Context(), "test-cache", "redis", "redis://"+mr.Addr())
	require.NoError(t, err) // Should succeed with miniredis

	// Named cache should now exist
	namedCache := registry.GetNamed("test-cache")
	assert.NotNil(t, namedCache)
	assert.Equal(t, cachefx.DialectRedis, namedCache.GetDialect())
}

func TestRegistry_LoadFromConfig(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)
	mr1 := setupTestRedisServer(t)
	mr2 := setupTestRedisServer(t)

	config := &cachefx.Config{
		Caches: map[string]cachefx.ConfigCache{
			"default": {
				Provider: "redis",
				DSN:      "redis://" + mr1.Addr(),
			},
			"secondary": {
				Provider: "redis",
				DSN:      "redis://" + mr2.Addr(),
			},
		},
	}

	err := registry.LoadFromConfig(t.Context(), config)
	require.NoError(t, err) // Should succeed with miniredis

	// Verify caches were added successfully
	assert.NotNil(t, registry.GetDefault())
	assert.NotNil(t, registry.GetNamed("secondary"))
	assert.Equal(t, cachefx.DialectRedis, registry.GetDefault().GetDialect())
	assert.Equal(t, cachefx.DialectRedis, registry.GetNamed("secondary").GetDialect())
}

func TestRegistry_AddConnection_InvalidDSN(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)

	// Test with invalid DSN
	err := registry.AddConnection(t.Context(), "invalid", "redis", "invalid-dsn")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add connection")

	// Cache should not be added
	assert.Nil(t, registry.GetNamed("invalid"))
}

func TestRegistry_AddConnection_ConnectionFailure(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)

	// Test with valid DSN but unreachable server
	err := registry.AddConnection(t.Context(), "unreachable", "redis", "redis://localhost:9999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add connection")

	// Cache should not be added
	assert.Nil(t, registry.GetNamed("unreachable"))
}
