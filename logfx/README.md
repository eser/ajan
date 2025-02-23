# ajan/logfx

## Overview

The **logfx** package is a configurable logging solution leverages the
`log/slog` of the standard library for structured logging. It includes
pretty-printing options and a fx module for the `ajan/di` package. The package
supports OpenTelemetry-compatible severity levels and provides extensive test
coverage to ensure reliability and correctness.

The documentation below provides an overview of the package, its types,
functions, and usage examples. For more detailed information, refer to the
source code and tests.

## Configuration

Configuration struct for the logger:

```go
type Config struct {
	Level      string `conf:"LEVEL"      default:"INFO"`
	PrettyMode bool   `conf:"PRETTY"     default:"true"`
	AddSource  bool   `conf:"ADD_SOURCE" default:"false"`
}
```

The supported log levels (in ascending order of severity) are:

- `TRACE` - Detailed information for debugging
- `DEBUG` - Debugging information
- `INFO` - General operational information
- `WARN` - Warning messages for potentially harmful situations
- `ERROR` - Error messages for serious problems
- `FATAL` - Critical errors causing program termination
- `PANIC` - Critical errors causing panic

These levels are compatible with OpenTelemetry Severity levels, allowing
seamless integration with observability platforms.

## API

### NewLogger function

Creates a new `logfx.Logger` object based on the provided configuration.

```go
// func NewLogger(w io.Writer, config *Config) (*logfx.Logger, error)

logger, err := logfx.NewLogger(os.Stdout, config)
```

### NewLoggerAsDefault function

Creates a new `logfx.Logger` object based on the provided configuration and
makes it default slog instance.

```go
// func NewLoggerAsDefault(w io.Writer, config *Config) (*logfx.Logger, error)

logger, err := logfx.NewLoggerAsDefault(os.Stdout, config)
```

### Colored outputs

Logs ANSI-colored strings.

```go
// func Colored(color Color, message string) string

// available colors:
//	ColorReset        ColorDimGray
//	ColorRed          ColorLightRed
//	ColorGreen        ColorLightGreen
//	ColorYellow       ColorLightYellow
//	ColorBlue         ColorLightBlue
//	ColorMagenta      ColorLightMagenta
//	ColorCyan         ColorLightCyan
//	ColorGray         ColorLightGray

logger.Fatal(
  logfx.Colored(logfx.ColorLightYellow, "Hello, World!"),
  slog.String("first_name", "Eser"),
  slog.String("last_name", "Ozvataf"),
)
```
