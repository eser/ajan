package httpclient

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	DefaultFailureThreshold     = 5
	DefaultResetTimeout         = 10 * time.Second
	DefaultHalfOpenSuccess      = 2
	DefaultServerErrorThreshold = 500
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrMaxRetries is returned when max retries are exceeded.
	ErrMaxRetries = errors.New("max retries exceeded")
	// ErrRequestBodyNotRetriable is returned when request body cannot be retried.
	ErrRequestBodyNotRetriable = errors.New(
		"request body cannot be retried, implement GetBody to enable retries",
	)
	// ErrAllRetryAttemptsFailed is returned when all retry attempts fail.
	ErrAllRetryAttemptsFailed = errors.New("all retry attempts failed")
	// ErrTransportError is returned when the underlying transport fails.
	ErrTransportError = errors.New("transport error")
	// ErrRequestContextError is returned when request context is cancelled.
	ErrRequestContextError = errors.New("request context error")
)

type ResilientTransport struct {
	transport      http.RoundTripper
	circuitBreaker *CircuitBreaker
	retryStrategy  *RetryStrategy
}

func NewResilientTransport(
	transport http.RoundTripper,
	cb *CircuitBreaker,
	rs *RetryStrategy,
) *ResilientTransport {
	if transport == nil {
		transport = http.DefaultTransport
	}

	if cb == nil {
		cb = NewCircuitBreaker(CircuitBreakerConfig{
			Enabled:               true,
			FailureThreshold:      DefaultFailureThreshold,
			ResetTimeout:          DefaultResetTimeout,
			HalfOpenSuccessNeeded: DefaultHalfOpenSuccess,
			ServerErrorThreshold:  DefaultServerErrorThreshold,
		})
	}

	if rs == nil {
		rs = DefaultRetryStrategy()
	}

	return &ResilientTransport{
		transport:      transport,
		circuitBreaker: cb,
		retryStrategy:  rs,
	}
}

func (t *ResilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.circuitBreaker.IsAllowed() {
		return nil, ErrCircuitOpen
	}

	if req.Body != nil && req.GetBody == nil {
		return nil, ErrRequestBodyNotRetriable
	}

	var lastErr error

	var resp *http.Response

	for attempt := range t.retryStrategy.Config.MaxAttempts {
		if attempt > 0 {
			var err error

			req, err = t.handleRetry(req, attempt)
			if err != nil {
				return nil, err
			}
		}

		resp, lastErr = t.handleRequest(req)
		if lastErr == nil && resp.StatusCode < t.circuitBreaker.Config.ServerErrorThreshold {
			return resp, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrAllRetryAttemptsFailed, lastErr)
	}

	return nil, ErrMaxRetries
}

// CancelRequest implements the optional CancelRequest method for http.RoundTripper.
func (t *ResilientTransport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(req *http.Request)
	}

	if cr, ok := t.transport.(canceler); ok {
		cr.CancelRequest(req)
	}
}

// handleRequest performs a single request attempt and handles the response.
func (t *ResilientTransport) handleRequest(req *http.Request) (*http.Response, error) {
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		t.circuitBreaker.OnFailure()

		if !t.circuitBreaker.IsAllowed() {
			return nil, ErrCircuitOpen
		}

		return nil, fmt.Errorf("%w: %w", ErrTransportError, err)
	}

	if resp.StatusCode >= t.circuitBreaker.Config.ServerErrorThreshold {
		t.circuitBreaker.OnFailure()

		if !t.circuitBreaker.IsAllowed() {
			return nil, ErrCircuitOpen
		}

		return resp, nil
	}

	t.circuitBreaker.OnSuccess()

	return resp, nil
}

// handleRetry manages the retry backoff and request cloning.
func (t *ResilientTransport) handleRetry(req *http.Request, attempt uint) (*http.Request, error) {
	backoff := t.retryStrategy.NextBackoff(attempt)
	if backoff <= 0 {
		return nil, ErrMaxRetries
	}

	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-req.Context().Done():
		return nil, fmt.Errorf("%w: %w", ErrRequestContextError, req.Context().Err())
	case <-timer.C:
	}

	return req.Clone(req.Context()), nil
}
