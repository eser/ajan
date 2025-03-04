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
		expectedErr string
	}{
		{
			name: "ValidConfig",
			config: &logfx.Config{
				Level:      "info",
				PrettyMode: true,
				AddSource:  true,
			},
			wantErr:     false,
			expectedErr: "",
		},
		{
			name: "InvalidLogLevel",
			config: &logfx.Config{
				Level:      "invalid",
				PrettyMode: true,
				AddSource:  true,
			},
			wantErr:     true,
			expectedErr: "failed to parse log level: unknown error level \"invalid\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger, err := logfx.NewLogger(os.Stdout, tt.config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, logger)
				assert.Equal(t, tt.expectedErr, err.Error())

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}
