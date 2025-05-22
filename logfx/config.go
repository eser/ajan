package logfx

type Config struct {
	Level      string `conf:"level"      default:"INFO"`
	PrettyMode bool   `conf:"pretty"     default:"true"`
	AddSource  bool   `conf:"add_source" default:"false"`
}
