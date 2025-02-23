package cachefx_test

import (
	"testing"

	"github.com/eser/ajan/cachefx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineDialect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		provider    string
		dsn         string
		wantDialect cachefx.Dialect
		wantErr     error
	}{
		{
			name:        "Redis provider explicit",
			provider:    "redis",
			dsn:         "redis://localhost:6379",
			wantDialect: cachefx.DialectRedis,
			wantErr:     nil,
		},
		{
			name:        "Redis DSN implicit",
			provider:    "",
			dsn:         "redis://localhost:6379",
			wantDialect: cachefx.DialectRedis,
			wantErr:     nil,
		},
		{
			name:        "Unknown provider",
			provider:    "unknown",
			dsn:         "redis://localhost:6379",
			wantDialect: "",
			wantErr:     cachefx.ErrUnknownProvider,
		},
		{
			name:        "Unable to determine dialect",
			provider:    "",
			dsn:         "invalid://localhost:6379",
			wantDialect: "",
			wantErr:     cachefx.ErrUnableToDetermineDialect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dialect, err := cachefx.DetermineDialect(tt.provider, tt.dsn)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantDialect, dialect)
		})
	}
}
