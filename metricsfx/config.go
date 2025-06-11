package metricsfx

import "time"

// Config holds configuration for metrics export.
type Config struct {
	// Service information
	ServiceName    string `conf:"service_name"    default:""`
	ServiceVersion string `conf:"service_version" default:""`

	// OpenTelemetry Collector configuration (preferred)
	OTLPEndpoint string `conf:"otlp_endpoint" default:""`

	// Legacy direct exporters (still supported)
	PrometheusEndpoint string `conf:"prometheus_endpoint" default:""`

	// Export interval
	ExportInterval time.Duration `conf:"export_interval" default:"30s"`

	RegisterNativeCollectors bool `conf:"register_native_collectors" default:"true"`

	OTLPInsecure bool `conf:"otlp_insecure" default:"true"`
}
