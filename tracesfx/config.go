package tracesfx

import "time"

// Config holds configuration for traces export.
type Config struct {
	// Service information
	ServiceName    string `conf:"service_name"    default:""`
	ServiceVersion string `conf:"service_version" default:""`

	// Connection name for OTLP export (uses connfx registry)
	OTLPConnectionName string `conf:"otlp_connection_name" default:""`

	// Sampling configuration
	SampleRatio float64 `conf:"sample_ratio" default:"1.0"`

	// Batch export configuration
	BatchTimeout time.Duration `conf:"batch_timeout" default:"5s"`
	BatchSize    int           `conf:"batch_size"    default:"512"`
}
