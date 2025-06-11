package tracesfx

import "time"

// Config holds configuration for traces export.
type Config struct {
	// Service information
	ServiceName    string `conf:"service_name"    default:""`
	ServiceVersion string `conf:"service_version" default:""`

	// OpenTelemetry Collector configuration
	OTLPEndpoint string `conf:"otlp_endpoint" default:""`
	OTLPInsecure bool   `conf:"otlp_insecure" default:"true"`

	// Sampling configuration
	SampleRatio float64 `conf:"sample_ratio" default:"1.0"`

	// Batch export configuration
	BatchTimeout time.Duration `conf:"batch_timeout" default:"5s"`
	BatchSize    int           `conf:"batch_size"    default:"512"`
}
