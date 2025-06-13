package logfx

type Config struct {
	Level string `conf:"level" default:"INFO"`

	// Connection name for OTLP export (uses connfx registry)
	OTLPConnectionName string `conf:"otlp_connection_name" default:""`

	DefaultLogger bool `conf:"default"    default:"false"`
	PrettyMode    bool `conf:"pretty"     default:"true"`
	AddSource     bool `conf:"add_source" default:"false"`
}
