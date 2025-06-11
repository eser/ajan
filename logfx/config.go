package logfx

type Config struct {
	Level string `conf:"level" default:"INFO"`

	// OpenTelemetry Collector configuration (preferred)
	OTLPEndpoint string `conf:"otlp_endpoint" default:""`

	// Direct Loki export (legacy/additional option)
	LokiURI    string `conf:"loki_uri"   default:""`
	LokiLabel  string `conf:"loki_label" default:""`
	PrettyMode bool   `conf:"pretty"     default:"true"`
	AddSource  bool   `conf:"add_source" default:"false"`

	OTLPInsecure bool `conf:"otlp_insecure" default:"true"`
}
