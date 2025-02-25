package httpclient

import (
	"time"
)

type Config struct {
	CircuitBreaker CircuitBreakerConfig `conf:"CIRCUIT_BREAKER"`
	Retry          RetryConfig          `conf:"RETRY"`
}

type CircuitBreakerConfig struct {
	Enabled               bool          `conf:"ENABLED"                  default:"true"`
	FailureThreshold      uint          `conf:"FAILURE_THRESHOLD"        default:"5"`
	ResetTimeout          time.Duration `conf:"RESET_TIMEOUT"            default:"10s"`
	HalfOpenSuccessNeeded uint          `conf:"HALF_OPEN_SUCCESS_NEEDED" default:"2"`
	ServerErrorThreshold  int           `conf:"SERVER_ERROR_THRESHOLD"   default:"500"`
}

type RetryConfig struct {
	Enabled         bool          `conf:"ENABLED"          default:"true"`
	MaxAttempts     uint          `conf:"MAX_ATTEMPTS"     default:"3"`
	InitialInterval time.Duration `conf:"INITIAL_INTERVAL" default:"100ms"`
	MaxInterval     time.Duration `conf:"MAX_INTERVAL"     default:"10s"`
	Multiplier      float64       `conf:"MULTIPLIER"       default:"2"`
	RandomFactor    float64       `conf:"RANDOM_FACTOR"    default:"0.1"`
}
