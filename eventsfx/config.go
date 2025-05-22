package eventsfx

import "time"

type Config struct {
	DefaultBufferSize int           `conf:"default_buffer_size" default:"100"`
	ReplyTimeout      time.Duration `conf:"reply_timeout"       default:"5s"`
}
