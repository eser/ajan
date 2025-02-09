package eventsfx

import "time"

type Config struct {
	DefaultBufferSize int           `conf:"DEFAULT_BUFFER_SIZE" default:"100"`
	ReplyTimeout      time.Duration `conf:"REPLY_TIMEOUT"       default:"5s"`
}
