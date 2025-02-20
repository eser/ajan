package datafx_test

import (
	"testing"

	"github.com/eser/ajan/datafx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineDialect(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name        string
		provider    string
		dsn         string
		want        datafx.Dialect
		wantErr     error
		errContains string
	}{
		{
			name:        "explicit postgres provider",
			provider:    "postgres",
			dsn:         "",
			want:        datafx.DialectPostgres,
			wantErr:     nil,
			errContains: "",
		},
		{
			name:        "explicit mysql provider",
			provider:    "mysql",
			dsn:         "",
			want:        datafx.DialectMySQL,
			wantErr:     nil,
			errContains: "",
		},
		{
			name:        "explicit sqlite provider",
			provider:    "sqlite",
			dsn:         "",
			want:        datafx.DialectSQLite,
			wantErr:     nil,
			errContains: "",
		},
		{
			name:        "unknown provider",
			provider:    "unknown",
			dsn:         "",
			want:        "",
			wantErr:     datafx.ErrUnknownProvider,
			errContains: "unknown provider - \"unknown\"",
		},
		{
			name:        "postgres dsn",
			provider:    "",
			dsn:         "postgres://user:pass@localhost:5432/dbname",
			want:        datafx.DialectPostgres,
			wantErr:     nil,
			errContains: "",
		},
		{
			name:        "mysql dsn",
			provider:    "",
			dsn:         "mysql://user:pass@localhost:3306/dbname",
			want:        datafx.DialectMySQL,
			wantErr:     nil,
			errContains: "",
		},
		{
			name:        "sqlite dsn",
			provider:    "",
			dsn:         "sqlite://path/to/db.sqlite",
			want:        datafx.DialectSQLite,
			wantErr:     nil,
			errContains: "",
		},
		{
			name:        "unrecognized dsn",
			provider:    "",
			dsn:         "invalid://connection-string",
			want:        "",
			wantErr:     datafx.ErrUnableToDetermineDialect,
			errContains: "unable to determine dialect",
		},
		{
			name:        "case insensitive dsn",
			provider:    "",
			dsn:         "POSTGRES://user:pass@localhost:5432/dbname",
			want:        datafx.DialectPostgres,
			wantErr:     nil,
			errContains: "",
		},
		{
			name:        "provider takes precedence over dsn",
			provider:    "mysql",
			dsn:         "postgres://user:pass@localhost:5432/dbname",
			want:        datafx.DialectMySQL,
			wantErr:     nil,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := datafx.DetermineDialect(tt.provider, tt.dsn)
			if tt.errContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
