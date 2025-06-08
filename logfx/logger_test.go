package logfx_test

import (
	"os"
	"testing"

	"github.com/eser/ajan/logfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *logfx.Config
		wantErr     bool
		expectedErr error
	}{
		{
			name: "ValidConfig",
			config: &logfx.Config{
				Level:      "info",
				PrettyMode: true,
				AddSource:  true,
			},
			wantErr:     false,
			expectedErr: nil,
		},
		{
			name: "InvalidLogLevel",
			config: &logfx.Config{
				Level:      "invalid",
				PrettyMode: true,
				AddSource:  true,
			},
			wantErr:     true,
			expectedErr: logfx.ErrFailedToParseLogLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger, err := logfx.NewLogger(os.Stdout, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				assert.Nil(t, logger)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}
