package httpclient

import (
	"net/http"
)

// Client is a drop-in replacement for http.Client with built-in circuit breaker and retry mechanisms.
type Client struct {
	*http.Client
	transport *ResilientTransport
}

// NewClient creates a new HTTP client with the specified circuit breaker and retry strategy.
func NewClient(cb *CircuitBreaker, rs *RetryStrategy) *Client {
	transport := NewResilientTransport(nil, cb, rs)

	return &Client{
		Client: &http.Client{ //nolint:exhaustruct
			Transport: transport,
		},
		transport: transport,
	}
}

// DefaultClient creates a new HTTP client with default circuit breaker and retry settings.
func DefaultClient() *Client {
	return NewClient(
		NewCircuitBreaker(CircuitBreakerConfig{
			Enabled:               true,
			FailureThreshold:      DefaultFailureThreshold,
			ResetTimeout:          DefaultResetTimeout,
			HalfOpenSuccessNeeded: DefaultHalfOpenSuccess,
			ServerErrorThreshold:  DefaultServerErrorThreshold,
		}),
		DefaultRetryStrategy(),
	)
}

// CircuitBreaker returns the underlying circuit breaker.
func (c *Client) CircuitBreaker() *CircuitBreaker {
	return c.transport.circuitBreaker
}

// RetryStrategy returns the underlying retry strategy.
func (c *Client) RetryStrategy() *RetryStrategy {
	return c.transport.retryStrategy
}
