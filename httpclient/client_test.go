package httpclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eser/ajan/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func closeBody(t *testing.T, resp *http.Response) {
	t.Helper()

	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}
}

func TestClientSuccessfulRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := httpclient.DefaultClient()

	ctx := t.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	defer closeBody(t, resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientCircuitBreaker(t *testing.T) {
	t.Parallel()

	failureCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failureCount++

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cb := httpclient.NewCircuitBreaker(3, time.Second, 1)
	client := httpclient.NewClient(cb, nil)
	ctx := t.Context()

	// Make requests until circuit breaker opens
	for range 4 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		defer closeBody(t, resp)

		if errors.Is(err, httpclient.ErrCircuitOpen) {
			assert.Equal(t, 3, failureCount)

			return
		}

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	}

	t.Error("circuit breaker did not open")
}

func TestClientRetryMechanism(t *testing.T) {
	t.Parallel()

	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rs := httpclient.NewRetryStrategy(3, time.Millisecond, time.Second, 1.0, 0)
	client := httpclient.NewClient(nil, rs)

	ctx := t.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	defer closeBody(t, resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, attemptCount)
}

func TestClientContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := httpclient.DefaultClient()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	defer closeBody(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), context.DeadlineExceeded.Error())
}
