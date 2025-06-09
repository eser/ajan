package adapters

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/eser/ajan/connfx"
)

const (
	DefaultHTTPTimeout = 30 * time.Second
	HealthCheckTimeout = 2 * time.Second
)

var (
	ErrFailedToCreateHTTPClient = errors.New("failed to create HTTP client")
	ErrFailedToHealthCheckHTTP  = errors.New("failed to health check HTTP endpoint")
	ErrInvalidConfigTypeHTTP    = errors.New("invalid config type for HTTP connection")
	ErrUnsupportedBodyType      = errors.New("unsupported body type")
	ErrFailedToCreateRequest    = errors.New("failed to create HTTP request")
)

// HTTPConnection represents an HTTP API connection.
type HTTPConnection struct {
	lastHealth time.Time
	client     *http.Client
	headers    map[string]string
	protocol   string
	baseURL    string
	state      int32 // atomic field for connection state
}

// HTTPConnectionFactory creates HTTP connections.
type HTTPConnectionFactory struct {
	protocol string
}

// NewHTTPConnectionFactory creates a new HTTP connection factory.
func NewHTTPConnectionFactory(protocol string) *HTTPConnectionFactory {
	return &HTTPConnectionFactory{
		protocol: protocol,
	}
}

func (f *HTTPConnectionFactory) CreateConnection(
	ctx context.Context,
	config *connfx.ConfigTarget,
) (connfx.Connection, error) {
	// Create HTTP client with configuration
	client, headers, err := f.buildHTTPClient(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateHTTPClient, err)
	}

	baseURL := config.URL

	// Initial health check
	conn := &HTTPConnection{
		protocol:   f.protocol,
		client:     client,
		baseURL:    baseURL,
		headers:    headers,
		state:      int32(connfx.ConnectionStateConnected),
		lastHealth: time.Time{},
	}

	// Perform initial health check
	status := conn.HealthCheck(ctx)
	if status.State == connfx.ConnectionStateError {
		return nil, fmt.Errorf("%w: %w", ErrFailedToHealthCheckHTTP, status.Error)
	}

	return conn, nil
}

func (f *HTTPConnectionFactory) GetProtocol() string {
	return f.protocol
}

func (f *HTTPConnectionFactory) GetSupportedBehaviors() []connfx.ConnectionBehavior {
	return []connfx.ConnectionBehavior{connfx.ConnectionBehaviorStateless}
}

func (f *HTTPConnectionFactory) buildHTTPClient( //nolint:cyclop
	config *connfx.ConfigTarget,
) (*http.Client, map[string]string, error) {
	// Configure transport
	transport := &http.Transport{} //nolint:exhaustruct

	// TLS configuration
	if config.TLS || config.TLSSkipVerify {
		transport.TLSClientConfig = &tls.Config{ //nolint:exhaustruct
			InsecureSkipVerify: config.TLSSkipVerify, //nolint:gosec
		}

		// Load client certificates if provided
		if config.CertFile != "" && config.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load client certificate: %w", err)
			}

			transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
		}
	}

	// Create HTTP client
	client := &http.Client{ //nolint:exhaustruct
		Transport: transport,
	}

	// Set timeout
	if config.Timeout > 0 {
		client.Timeout = config.Timeout
	} else {
		client.Timeout = DefaultHTTPTimeout
	}

	// Build default headers
	headers := make(map[string]string)
	headers["User-Agent"] = "connfx-http-client/1.0"

	// Add custom headers from properties
	if config.Properties != nil {
		if customHeaders, ok := config.Properties["headers"].(map[string]any); ok {
			for k, v := range customHeaders {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
		}
	}

	return client, headers, nil
}

// Connection interface implementation

func (c *HTTPConnection) GetBehaviors() []connfx.ConnectionBehavior {
	return []connfx.ConnectionBehavior{connfx.ConnectionBehaviorStateless}
}

func (c *HTTPConnection) GetProtocol() string {
	return c.protocol
}

func (c *HTTPConnection) GetState() connfx.ConnectionState {
	state := atomic.LoadInt32(&c.state)

	return connfx.ConnectionState(state)
}

func (c *HTTPConnection) HealthCheck( //nolint:cyclop
	ctx context.Context,
) *connfx.HealthStatus {
	start := time.Now()
	status := &connfx.HealthStatus{ //nolint:exhaustruct
		Timestamp: start,
	}

	// Create health check request (HEAD request to base URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL, nil)
	if err != nil {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateError))
		status.State = connfx.ConnectionStateError
		status.Error = err
		status.Message = fmt.Sprintf("Failed to create health check request: %v", err)
		status.Latency = time.Since(start)

		return status
	}

	// Add default headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Perform request
	resp, err := c.client.Do(req)
	status.Latency = time.Since(start)

	if err != nil {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateError))
		status.State = connfx.ConnectionStateError
		status.Error = err
		status.Message = fmt.Sprintf("Health check failed: %v", err)

		return status
	}

	defer func() {
		_ = resp.Body.Close() // Ignore close error for health check
	}()

	// Check response status using switch instead of if-else chain
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateConnected))
		status.State = connfx.ConnectionStateConnected
		status.Message = fmt.Sprintf("Connected (status=%d)", resp.StatusCode)
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// Try GET request if HEAD fails with 405
		if resp.StatusCode == http.StatusMethodNotAllowed {
			if getStatus := c.tryGetRequest(ctx, start); getStatus != nil {
				return getStatus
			}
		}

		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateError))
		status.State = connfx.ConnectionStateError
		status.Message = fmt.Sprintf("Client error (status=%d)", resp.StatusCode)
	default:
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateError))
		status.State = connfx.ConnectionStateError
		status.Message = fmt.Sprintf("Server error (status=%d)", resp.StatusCode)
	}

	c.lastHealth = start

	return status
}

func (c *HTTPConnection) Close(ctx context.Context) error {
	atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateDisconnected))
	// HTTP clients don't need explicit closing, but we can close idle connections
	if transport, ok := c.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

func (c *HTTPConnection) GetRawConnection() any {
	return c.client
}

// Additional HTTP-specific methods

// GetClient returns the underlying HTTP client.
func (c *HTTPConnection) GetClient() *http.Client {
	return c.client
}

// GetBaseURL returns the base URL for this connection.
func (c *HTTPConnection) GetBaseURL() string {
	return c.baseURL
}

// GetHeaders returns the default headers for this connection.
func (c *HTTPConnection) GetHeaders() map[string]string {
	headers := make(map[string]string)
	maps.Copy(headers, c.headers)

	return headers
}

// NewRequest creates a new HTTP request with the connection's default headers.
func (c *HTTPConnection) NewRequest(
	ctx context.Context,
	method string,
	path string,
	body any,
) (*http.Request, error) {
	url := c.baseURL

	if path != "" {
		if path[0] != '/' {
			url += "/"
		}

		url += path
	}

	var req *http.Request

	var err error

	// Handle different body types
	switch v := body.(type) {
	case nil:
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	case string:
		req, err = http.NewRequestWithContext(ctx, method, url, strings.NewReader(v))
	case []byte:
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(v))
	case io.Reader:
		req, err = http.NewRequestWithContext(ctx, method, url, v)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedBodyType, body)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToCreateRequest, err)
	}

	// Add default headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// tryGetRequest attempts a GET request when HEAD fails with 405.
func (c *HTTPConnection) tryGetRequest(ctx context.Context, start time.Time) *connfx.HealthStatus {
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil
	}

	for k, v := range c.headers {
		getReq.Header.Set(k, v)
	}

	getResp, err := c.client.Do(getReq)
	if err != nil {
		return nil
	}

	defer func() {
		_ = getResp.Body.Close() // Ignore close error for health check
	}()

	if getResp.StatusCode >= 200 && getResp.StatusCode < 300 {
		atomic.StoreInt32(&c.state, int32(connfx.ConnectionStateConnected))
		c.lastHealth = start

		return &connfx.HealthStatus{ //nolint:exhaustruct
			Timestamp: start,
			State:     connfx.ConnectionStateConnected,
			Message:   fmt.Sprintf("Connected (status=%d)", getResp.StatusCode),
			Latency:   time.Since(start),
		}
	}

	return nil
}
