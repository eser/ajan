package cachefx_test

import (
	"os"
	"testing"

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

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)
	assert.NotNil(t, registry)
}

func TestRegistry_GetDefault(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)
	assert.Nil(t, registry.GetDefault())

	// Add a default cache
	err := registry.AddConnection(t.Context(), cachefx.DefaultCache, "redis", "redis://localhost:6379")
	require.Error(t, err) // Expected error since Redis is not running

	// Even with error, default should still be nil
	assert.Nil(t, registry.GetDefault())
}

func TestRegistry_GetNamed(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)

	// Test non-existent cache
	assert.Nil(t, registry.GetNamed("non-existent"))

	// Add a named cache
	err := registry.AddConnection(t.Context(), "test-cache", "redis", "redis://localhost:6379")
	require.Error(t, err) // Expected error since Redis is not running

	// Even with error, named cache should still be nil
	assert.Nil(t, registry.GetNamed("test-cache"))
}

func TestRegistry_LoadFromConfig(t *testing.T) {
	t.Parallel()

	registry := setupTestRegistry(t)

	config := &cachefx.Config{
		Caches: map[string]cachefx.ConfigCache{
			"default": {
				Provider: "redis",
				DSN:      "redis://localhost:6379",
			},
			"secondary": {
				Provider: "redis",
				DSN:      "redis://localhost:6380",
			},
		},
	}

	err := registry.LoadFromConfig(t.Context(), config)
	require.Error(t, err) // Expected error since Redis is not running

	// Verify no caches were added due to connection errors
	assert.Nil(t, registry.GetDefault())
	assert.Nil(t, registry.GetNamed("secondary"))
}
