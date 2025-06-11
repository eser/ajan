package logfx_test

import (
	"os"
	"testing"

	"github.com/eser/ajan/logfx"
	"github.com/stretchr/testify/assert"
)

func TestRegisterLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *logfx.Config
	}{
		{
			name: "ValidConfig",
			config: &logfx.Config{
				Level:        "info",
				PrettyMode:   true,
				AddSource:    true,
				OTLPEndpoint: "",
				OTLPInsecure: false,
				LokiURI:      "",
				LokiLabel:    "",
			},
		},
		{
			name: "InvalidLogLevel",
			config: &logfx.Config{
				Level:        "invalid",
				PrettyMode:   true,
				AddSource:    true,
				OTLPEndpoint: "",
				OTLPInsecure: false,
				LokiURI:      "",
				LokiLabel:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := logfx.NewLogger(os.Stdout, tt.config)
			assert.NotNil(t, logger)
		})
	}
}
