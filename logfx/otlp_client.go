package logfx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrFailedToCreateOTLPLogExporter = errors.New("failed to create OTLP log exporter")
	ErrFailedToCreateLogProcessor    = errors.New("failed to create log processor")
	ErrOTLPNotConfigured             = errors.New("OTLP not configured")
	ErrFailedToShutdownOTLP          = errors.New("failed to shutdown OTLP logger provider")
)

// OTLPClient handles sending logs to OpenTelemetry collector.
type OTLPClient struct {
	loggerProvider *sdklog.LoggerProvider
	logger         log.Logger
}

// NewOTLPClient creates a new OTLP client for sending logs to OpenTelemetry collector.
func NewOTLPClient(endpoint string, insecure bool) (*OTLPClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("%w (endpoint is empty)", ErrOTLPNotConfigured)
	}

	// Create OTLP log exporter
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(endpoint),
	}

	if insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	exporter, err := otlploghttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf(
			"%w (endpoint=%q): %w",
			ErrFailedToCreateOTLPLogExporter,
			endpoint,
			err,
		)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("logfx-service"), // This could be configurable
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	// Create log processor
	processor := sdklog.NewBatchProcessor(exporter)

	// Create logger provider
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)

	// Get logger
	logger := loggerProvider.Logger("logfx")

	return &OTLPClient{
		loggerProvider: loggerProvider,
		logger:         logger,
	}, nil
}

// SendLog sends a log record to OpenTelemetry collector asynchronously.
func (c *OTLPClient) SendLog(ctx context.Context, rec slog.Record) {
	go func() {
		if err := c.sendLogSync(ctx, rec); err != nil {
			// Use slog for error logging to avoid infinite recursion
			slog.Error("Failed to send log to OTLP collector", "error", err)
		}
	}()
}

// Shutdown gracefully shuts down the OTLP client.
func (c *OTLPClient) Shutdown(ctx context.Context) error {
	if c.loggerProvider != nil {
		if err := c.loggerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("%w: %w", ErrFailedToShutdownOTLP, err)
		}
	}

	return nil
}

// sendLogSync sends a log record to OpenTelemetry collector synchronously.
// Note: Always returns nil since OpenTelemetry's Emit method doesn't return errors,
// but kept for interface consistency with loki_client.go.
func (c *OTLPClient) sendLogSync(ctx context.Context, rec slog.Record) error { //nolint:unparam
	// Convert slog level to OpenTelemetry log severity
	severity := convertSlogLevel(rec.Level)

	// Build attributes
	var attrs []log.KeyValue

	// Add OpenTelemetry trace context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		attrs = append(attrs,
			log.String("trace_id", spanCtx.TraceID().String()),
			log.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add HTTP correlation ID if available (for HTTP request correlation)
	if correlationID := getCorrelationIDFromContext(ctx); correlationID != "" {
		attrs = append(attrs, log.String("correlation_id", correlationID))
	}

	// Add attributes from the record
	rec.Attrs(func(attr slog.Attr) bool {
		otlpAttr := convertSlogAttribute(attr)
		if otlpAttr != nil {
			attrs = append(attrs, *otlpAttr)
		}

		return true
	})

	// Create log record
	logRecord := log.Record{}
	logRecord.SetTimestamp(rec.Time)
	logRecord.SetBody(log.StringValue(rec.Message))
	logRecord.SetSeverity(severity)
	logRecord.SetSeverityText(rec.Level.String())

	// Add attributes to the record
	for _, attr := range attrs {
		logRecord.AddAttributes(attr)
	}

	// Emit the log record
	c.logger.Emit(ctx, logRecord)

	return nil
}

// convertSlogLevel converts slog.Level to OpenTelemetry log.Severity.
func convertSlogLevel(level slog.Level) log.Severity {
	switch level {
	case LevelTrace:
		return log.SeverityTrace
	case LevelDebug:
		return log.SeverityDebug
	case LevelInfo:
		return log.SeverityInfo
	case LevelWarn:
		return log.SeverityWarn
	case LevelError:
		return log.SeverityError
	case LevelFatal:
		return log.SeverityFatal
	case LevelPanic:
		return log.SeverityFatal4
	default:
		return log.SeverityInfo
	}
}

// convertSlogAttribute converts slog.Attr to OpenTelemetry log.KeyValue.
func convertSlogAttribute(attr slog.Attr) *log.KeyValue {
	key := attr.Key
	value := attr.Value

	switch value.Kind() {
	case slog.KindString, slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool:
		return convertBasicSlogValue(key, value)
	case slog.KindTime, slog.KindDuration:
		return convertTimeSlogValue(key, value)
	case slog.KindAny, slog.KindGroup, slog.KindLogValuer:
		return convertComplexSlogValue(key, value)
	default:
		return convertDefaultSlogValue(key, value)
	}
}

// convertBasicSlogValue converts basic slog values (string, int, bool, etc.)
func convertBasicSlogValue(key string, value slog.Value) *log.KeyValue {
	switch value.Kind() {
	case slog.KindString:
		kv := log.String(key, value.String())

		return &kv
	case slog.KindInt64:
		kv := log.Int64(key, value.Int64())

		return &kv
	case slog.KindUint64:
		// Convert uint64 to int64 safely, capping at max int64 value
		val := value.Uint64()
		if val > math.MaxInt64 {
			kv := log.String(key, strconv.FormatUint(val, 10))

			return &kv
		}

		kv := log.Int64(key, int64(val))

		return &kv
	case slog.KindFloat64:
		kv := log.Float64(key, value.Float64())

		return &kv
	case slog.KindBool:
		kv := log.Bool(key, value.Bool())

		return &kv
	case slog.KindTime, slog.KindDuration, slog.KindAny, slog.KindGroup, slog.KindLogValuer:
		// These are handled by other functions
		return nil
	default:
		return nil
	}
}

// convertTimeSlogValue converts time-related slog values.
func convertTimeSlogValue(key string, value slog.Value) *log.KeyValue {
	switch value.Kind() {
	case slog.KindTime:
		kv := log.String(key, value.Time().Format(time.RFC3339Nano))

		return &kv
	case slog.KindDuration:
		kv := log.String(key, value.Duration().String())

		return &kv
	case slog.KindString,
		slog.KindInt64,
		slog.KindUint64,
		slog.KindFloat64,
		slog.KindBool,
		slog.KindAny,
		slog.KindGroup,
		slog.KindLogValuer:
		// These are handled by other functions
		return nil
	default:
		return nil
	}
}

// convertComplexSlogValue converts complex slog values to string.
func convertComplexSlogValue(key string, value slog.Value) *log.KeyValue {
	kv := log.String(key, value.String())

	return &kv
}

// convertDefaultSlogValue converts any other slog values to string.
func convertDefaultSlogValue(key string, value slog.Value) *log.KeyValue {
	kv := log.String(key, value.String())

	return &kv
}
