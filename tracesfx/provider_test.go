package tracesfx_test

import (
	"testing"
	"time"

	"github.com/eser/ajan/tracesfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracesProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         *tracesfx.Config
		expectProvider bool
	}{
		{
			name: "no_otlp_endpoint",
			config: &tracesfx.Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				OTLPEndpoint:   "",
				OTLPInsecure:   false,
				SampleRatio:    1.0,
				BatchTimeout:   0,
				BatchSize:      0,
			},
			expectProvider: true,
		},
		{
			name: "with_otlp_endpoint",
			config: &tracesfx.Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				OTLPEndpoint:   "http://localhost:4318",
				OTLPInsecure:   true,
				SampleRatio:    1.0,
				BatchTimeout:   5 * time.Second,
				BatchSize:      512,
			},
			expectProvider: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testTracesProvider(t, tt.config, tt.expectProvider)
		})
	}
}

func testTracesProvider(t *testing.T, config *tracesfx.Config, expectProvider bool) {
	t.Helper()

	provider := tracesfx.NewTracesProvider(config)
	require.NotNil(t, provider)

	err := provider.Init()
	if expectProvider {
		require.NoError(t, err)

		// Test getting a tracer
		tracer := provider.Tracer("test-tracer")
		assert.NotNil(t, tracer)

		// Test creating a span
		ctx := t.Context()
		spanCtx, span := tracer.Start(ctx, "test-span")
		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)

		// Test span operations
		span.SetAttributes()
		span.AddEvent("test-event")
		span.End()
	} else {
		require.Error(t, err)
	}

	// Cleanup
	err = provider.Shutdown(t.Context())
	require.NoError(t, err)
}

func TestCorrelationIntegration(t *testing.T) {
	t.Parallel()

	provider := tracesfx.NewTracesProvider(&tracesfx.Config{
		ServiceName:    "test-service",
		ServiceVersion: "",
		OTLPEndpoint:   "", // No-op for testing
		OTLPInsecure:   false,
		SampleRatio:    1.0,
		BatchTimeout:   0,
		BatchSize:      0,
	})

	err := provider.Init()
	require.NoError(t, err)

	defer func() {
		err := provider.Shutdown(t.Context())
		assert.NoError(t, err)
	}()

	tracer := provider.Tracer("test")

	// Test correlation ID integration
	ctx := t.Context()
	correlationID := "test-correlation-123"
	ctx = tracesfx.SetCorrelationIDInContext(ctx, correlationID)

	// Start span with correlation
	spanCtx, span := tracer.StartSpanWithCorrelation(ctx, "test-span")
	assert.NotNil(t, spanCtx)
	assert.NotNil(t, span)

	// Verify correlation ID can be retrieved
	retrievedID := tracesfx.GetCorrelationIDFromContext(spanCtx)
	assert.Equal(t, correlationID, retrievedID)

	span.End()
}
