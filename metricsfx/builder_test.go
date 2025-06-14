package metricsfx_test

import (
	"testing"
	"time"

	"github.com/eser/ajan/metricsfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMetricsProvider(t *testing.T) *metricsfx.MetricsProvider {
	t.Helper()

	provider := metricsfx.NewMetricsProvider(&metricsfx.Config{
		ServiceName:                   "",
		ServiceVersion:                "",
		OTLPConnectionName:            "", // No connection for testing
		ExportInterval:                30 * time.Second,
		NoNativeCollectorRegistration: true,
	}, nil) // nil registry for testing

	err := provider.Init()
	require.NoError(t, err)

	t.Cleanup(func() {
		err := provider.Shutdown(t.Context())
		if err != nil {
			t.Logf("Error shutting down metrics provider: %v", err)
		}
	})

	return provider
}

func TestMetricsBuilder_Counter(t *testing.T) {
	t.Parallel()

	provider := setupMetricsProvider(t)
	builder := provider.NewBuilder()

	counter, err := builder.Counter(
		"test_counter",
		"A test counter metric",
	).WithUnit("requests").Build()

	require.NoError(t, err)
	assert.NotNil(t, counter)

	// Test that we can use the counter
	ctx := t.Context()
	counter.Inc(ctx, metricsfx.StringAttr("method", "GET"))
	counter.Add(ctx, 5, metricsfx.StringAttr("method", "POST"))
}

func TestMetricsBuilder_Gauge(t *testing.T) {
	t.Parallel()

	provider := setupMetricsProvider(t)
	builder := provider.NewBuilder()

	gauge, err := builder.Gauge(
		"test_gauge",
		"A test gauge metric",
	).WithUnit("connections").Build()

	require.NoError(t, err)
	assert.NotNil(t, gauge)

	// Test that we can use the gauge
	ctx := t.Context()
	gauge.Set(ctx, 42, metricsfx.StringAttr("pool", "main"))
	gauge.SetBool(ctx, true, metricsfx.StringAttr("status", "healthy"))
}

func TestMetricsBuilder_Histogram(t *testing.T) {
	t.Parallel()

	provider := setupMetricsProvider(t)
	builder := provider.NewBuilder()

	histogram, err := builder.Histogram(
		"test_histogram",
		"A test histogram metric",
	).WithDurationBuckets().Build()

	require.NoError(t, err)
	assert.NotNil(t, histogram)

	// Test that we can use the histogram
	ctx := t.Context()
	histogram.Record(ctx, 1.5, metricsfx.StringAttr("operation", "read"))
	histogram.RecordDuration(ctx, 250*time.Millisecond, metricsfx.StringAttr("operation", "write"))
}

func TestMetricsBuilder_CustomBuckets(t *testing.T) {
	t.Parallel()

	provider := setupMetricsProvider(t)
	builder := provider.NewBuilder()

	histogram, err := builder.Histogram(
		"custom_buckets_histogram",
		"A histogram with custom buckets",
	).WithBuckets(0.1, 0.5, 1.0, 2.0, 5.0).Build()

	require.NoError(t, err)
	assert.NotNil(t, histogram)

	// Test that we can use the histogram
	ctx := t.Context()
	histogram.Record(ctx, 0.3, metricsfx.StringAttr("test", "value"))
}

func TestWorkerMetrics_Integration(t *testing.T) {
	t.Parallel()

	provider := setupMetricsProvider(t)
	builder := provider.NewBuilder()

	// Create some sample worker metrics
	workerTicks, err := builder.Counter(
		"worker_ticks_total",
		"Total number of worker ticks",
	).WithUnit("{tick}").Build()
	require.NoError(t, err)

	workerErrors, err := builder.Counter(
		"worker_errors_total",
		"Total number of worker errors",
	).WithUnit("{error}").Build()
	require.NoError(t, err)

	processingTime, err := builder.Histogram(
		"worker_processing_time_seconds",
		"Worker processing time in seconds",
	).WithDurationBuckets().Build()
	require.NoError(t, err)

	ctx := t.Context()

	// Test worker tick recording
	workerAttrs := metricsfx.WorkerAttrs("test-worker")
	workerTicks.Inc(ctx, workerAttrs...)

	// Test worker error recording
	testErr := assert.AnError
	errorAttrs := metricsfx.WorkerErrorAttrs("test-worker", testErr)
	workerErrors.Inc(ctx, errorAttrs...)

	// Test processing time recording
	processingTime.RecordDuration(ctx, 100*time.Millisecond, workerAttrs...)
}

func TestAttributeHelpers(t *testing.T) {
	t.Parallel()

	// Test WorkerAttrs
	workerAttrs := metricsfx.WorkerAttrs("test-worker")
	assert.Len(t, workerAttrs, 1)
	assert.Equal(t, "worker_name", string(workerAttrs[0].Key))
	assert.Equal(t, "test-worker", workerAttrs[0].Value.AsString())

	// Test ErrorAttrs
	testErr := assert.AnError
	errorAttrs := metricsfx.ErrorAttrs(testErr)
	assert.Len(t, errorAttrs, 1)
	assert.Equal(t, "error_type", string(errorAttrs[0].Key))

	// Test WorkerErrorAttrs
	workerErrorAttrs := metricsfx.WorkerErrorAttrs("test-worker", testErr)
	assert.Len(t, workerErrorAttrs, 2)

	// Test StatusAttrs
	statusAttrs := metricsfx.StatusAttrs("active")
	assert.Len(t, statusAttrs, 1)
	assert.Equal(t, "status", string(statusAttrs[0].Key))
	assert.Equal(t, "active", statusAttrs[0].Value.AsString())

	// Test TypeAttrs
	typeAttrs := metricsfx.TypeAttrs("batch_processing")
	assert.Len(t, typeAttrs, 1)
	assert.Equal(t, "type", string(typeAttrs[0].Key))
	assert.Equal(t, "batch_processing", typeAttrs[0].Value.AsString())
}

func TestMetricsBuilder_Build(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name             string
		config           *metricsfx.Config
		expectedError    bool
		expectedProvider bool
	}{
		{
			name: "valid config without OTLP",
			config: &metricsfx.Config{
				ServiceName:                   "test-service",
				ServiceVersion:                "1.0.0",
				OTLPConnectionName:            "", // No connection
				ExportInterval:                30 * time.Second,
				NoNativeCollectorRegistration: true,
			},
			expectedError:    false,
			expectedProvider: true,
		},
		{
			name: "valid config with connection name",
			config: &metricsfx.Config{
				ServiceName:                   "test-service",
				ServiceVersion:                "1.0.0",
				OTLPConnectionName:            "otlp-connection", // Connection configured
				ExportInterval:                30 * time.Second,
				NoNativeCollectorRegistration: true,
			},
			expectedError:    true, // Error expected because no registry provided
			expectedProvider: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create metrics provider with nil registry (for testing)
			provider := metricsfx.NewMetricsProvider(tt.config, nil)
			require.NotNil(t, provider)

			// Initialize the provider
			err := provider.Init()

			// If connection name is provided but no registry, expect error
			if tt.config.OTLPConnectionName != "" {
				require.Error(t, err)

				return // Skip builder test since init failed
			}

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.expectedProvider {
				assert.NotNil(t, provider)
			}

			// Test builder only if init succeeded
			if err == nil {
				builder := provider.NewBuilder()
				assert.NotNil(t, builder)
			}

			// Test cleanup
			ctx := t.Context()
			shutdownErr := provider.Shutdown(ctx)
			assert.NoError(t, shutdownErr)
		})
	}
}
