package httpclient

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

const (
	DefaultMaxAttempts     = 3
	DefaultInitialInterval = 100 * time.Millisecond
	DefaultMaxInterval     = 10 * time.Second
	DefaultMultiplier      = 2.0
	DefaultRandomFactor    = 0.1
	randomNumberRange      = 1000 // Range for random number generation in jitter calculation
)

type RetryStrategy struct {
	MaxAttempts     uint
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	RandomFactor    float64
}

// NewRetryStrategy creates a new retry strategy with the specified parameters.
func NewRetryStrategy(
	maxAttempts uint,
	initialInterval, maxInterval time.Duration,
	multiplier, randomFactor float64,
) *RetryStrategy {
	return &RetryStrategy{
		MaxAttempts:     maxAttempts,
		InitialInterval: initialInterval,
		MaxInterval:     maxInterval,
		Multiplier:      multiplier,
		RandomFactor:    randomFactor,
	}
}

func (r *RetryStrategy) NextBackoff(attempt uint) time.Duration {
	if attempt >= r.MaxAttempts {
		return 0
	}

	// Calculate exponential backoff
	backoff := float64(r.InitialInterval) * math.Pow(r.Multiplier, float64(attempt))

	// Apply random factor
	if r.RandomFactor > 0 {
		// Use crypto/rand for secure random number generation
		n, err := rand.Int(rand.Reader, big.NewInt(randomNumberRange))
		if err != nil {
			// Fallback to no jitter if random generation fails
			return time.Duration(backoff)
		}

		random := 1 + r.RandomFactor*(2*float64(n.Int64())/float64(randomNumberRange)-1)
		backoff *= random
	}

	// Ensure we don't exceed max interval
	if backoff > float64(r.MaxInterval) {
		backoff = float64(r.MaxInterval)
	}

	return time.Duration(backoff)
}

func DefaultRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxAttempts:     DefaultMaxAttempts,
		InitialInterval: DefaultInitialInterval,
		MaxInterval:     DefaultMaxInterval,
		Multiplier:      DefaultMultiplier,
		RandomFactor:    DefaultRandomFactor,
	}
}
