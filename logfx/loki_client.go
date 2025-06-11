package logfx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const (
	// HTTP client timeout for Loki requests.
	lokiHTTPTimeout = 10 * time.Second
	// Number of parts expected when splitting key=value pairs.
	keyValueParts = 2
)

var (
	ErrFailedToSendToLoki  = errors.New("failed to send log to loki")
	ErrFailedToMarshalLoki = errors.New("failed to marshal loki payload")
	ErrInvalidLokiResponse = errors.New("invalid loki response")
	ErrLokiNotConfigured   = errors.New("loki not configured")
)

// LokiStream represents a single Loki log stream.
type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// LokiPayload represents the payload sent to Loki.
type LokiPayload struct {
	Streams []LokiStream `json:"streams"`
}

// LokiClient handles sending logs to Loki.
type LokiClient struct {
	baseLabels map[string]string
	httpClient *http.Client
	uri        string
}

// NewLokiClient creates a new Loki client.
func NewLokiClient(uri, labelStr string) (*LokiClient, error) {
	if uri == "" {
		return nil, fmt.Errorf("%w (uri is empty)", ErrLokiNotConfigured)
	}

	baseLabels := parseLokiLabels(labelStr)

	return &LokiClient{
		uri:        uri,
		baseLabels: baseLabels,
		httpClient: &http.Client{
			Timeout:       lokiHTTPTimeout,
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
		},
	}, nil
}

// SendLog sends a log record to Loki asynchronously.
func (c *LokiClient) SendLog(ctx context.Context, rec slog.Record) {
	go func() {
		if err := c.sendLogSync(ctx, rec); err != nil {
			// Use slog for error logging to avoid infinite recursion
			slog.Error("Failed to send log to Loki", "error", err)
		}
	}()
}

// sendLogSync sends a log record to Loki synchronously.
func (c *LokiClient) sendLogSync(ctx context.Context, rec slog.Record) error {
	labels := c.buildLabels(ctx, rec)
	logData := c.buildLogData(ctx, rec)

	// Marshal log data to JSON
	logLine, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("%w (data=%+v): %w", ErrFailedToMarshalLoki, logData, err)
	}

	payload := c.createLokiPayload(labels, logLine, rec)

	return c.sendPayload(ctx, payload)
}

// buildLabels creates the labels map for the log entry.
func (c *LokiClient) buildLabels(ctx context.Context, rec slog.Record) map[string]string {
	labels := make(map[string]string)
	maps.Copy(labels, c.baseLabels)

	// Add standard labels
	labels["level"] = rec.Level.String()

	// Add OpenTelemetry trace context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		labels["trace_id"] = spanCtx.TraceID().String()
		labels["span_id"] = spanCtx.SpanID().String()
	}

	// Add HTTP correlation ID if available (for HTTP request correlation)
	if correlationID := getCorrelationIDFromContext(ctx); correlationID != "" {
		labels["correlation_id"] = correlationID
	}

	return labels
}

// buildLogData creates the log data map for the log entry.
func (c *LokiClient) buildLogData(ctx context.Context, rec slog.Record) map[string]any {
	logData := make(map[string]any)
	logData["msg"] = rec.Message
	logData["time"] = rec.Time.Format(time.RFC3339Nano)
	logData["level"] = rec.Level.String()

	// Add correlation ID to log data as well for easier querying
	if correlationID := getCorrelationIDFromContext(ctx); correlationID != "" {
		logData["correlation_id"] = correlationID
	}

	// Add attributes from the record
	rec.Attrs(func(attr slog.Attr) bool {
		logData[attr.Key] = attr.Value.Any()

		return true
	})

	return logData
}

// createLokiPayload creates the Loki payload structure.
func (c *LokiClient) createLokiPayload(
	labels map[string]string,
	logLine []byte,
	rec slog.Record,
) LokiPayload {
	return LokiPayload{
		Streams: []LokiStream{
			{
				Stream: labels,
				Values: [][]string{
					{
						// Loki expects timestamp in nanoseconds as string
						strconv.FormatInt(rec.Time.UnixNano(), 10),
						string(logLine),
					},
				},
			},
		},
	}
}

// sendPayload sends the payload to Loki.
func (c *LokiClient) sendPayload(ctx context.Context, payload LokiPayload) error {
	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w (payload=%+v): %w", ErrFailedToMarshalLoki, payload, err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.uri,
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		return fmt.Errorf("%w (uri=%q): %w", ErrFailedToSendToLoki, c.uri, err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w (uri=%q): %w", ErrFailedToSendToLoki, c.uri, err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Error("Failed to close response body", "error", closeErr)
		}
	}()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w (uri=%q, status=%d)", ErrInvalidLokiResponse, c.uri, resp.StatusCode)
	}

	return nil
}

// parseLokiLabels parses a comma-separated string of key=value pairs into a map.
func parseLokiLabels(labelStr string) map[string]string {
	labels := make(map[string]string)

	if labelStr == "" {
		return labels
	}

	pairs := strings.SplitSeq(labelStr, ",")
	for pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", keyValueParts)
		if len(parts) == keyValueParts {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if key != "" {
				labels[key] = value
			}
		}
	}

	return labels
}
