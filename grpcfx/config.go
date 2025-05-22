package grpcfx

import (
	"time"
)

type Config struct {
	Addr                    string        `conf:"addr"             default:":9090"`
	Reflection              bool          `conf:"reflection"       default:"true"`
	InitializationTimeout   time.Duration `conf:"init_timeout"     default:"25s"`
	GracefulShutdownTimeout time.Duration `conf:"shutdown_timeout" default:"5s"`
}
