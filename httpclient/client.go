package httpclient

import (
	"net/http"
)

// Client is a drop-in replacement for http.Client with built-in circuit breaker and retry mechanisms.
type Client struct {
	*http.Client

	Config    *Config
	Transport *ResilientTransport
}

// NewClient creates a new http client with the specified circuit breaker and retry strategy.
func NewClient(options ...NewClientOption) *Client {
	client := &Client{
		Client: nil,

		Config: &Config{
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:               true,
				FailureThreshold:      DefaultFailureThreshold,
				ResetTimeout:          DefaultResetTimeout,
				HalfOpenSuccessNeeded: DefaultHalfOpenSuccess,
			},
			RetryStrategy: RetryStrategyConfig{
				Enabled:         true,
				MaxAttempts:     DefaultMaxAttempts,
				InitialInterval: DefaultInitialInterval,
				MaxInterval:     DefaultMaxInterval,
				Multiplier:      DefaultMultiplier,
				RandomFactor:    DefaultRandomFactor,
			},

			ServerErrorThreshold: DefaultServerErrorThreshold,
		},
		Transport: nil,
	}

	for _, option := range options {
		option(client)
	}

	if client.Transport == nil {
		transport := NewResilientTransport(
			http.DefaultTransport,
			client.Config,
		)

		client.Transport = transport
	}

	client.Client = &http.Client{ //nolint:exhaustruct
		Transport: client.Transport,
	}

	return client
}
