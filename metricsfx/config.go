package metricsfx

import "time"

// Config holds configuration for metrics export.
type Config struct {
	// Service information
	ServiceName    string `conf:"service_name"    default:""`
	ServiceVersion string `conf:"service_version" default:""`

	// Connection name for OTLP export (uses connfx registry)
	OTLPConnectionName string `conf:"otlp_connection_name" default:""`

	// Export interval
	ExportInterval time.Duration `conf:"export_interval" default:"30s"`

	NoNativeCollectorRegistration bool `conf:"no_native_collector_registration" default:"false"`
}
