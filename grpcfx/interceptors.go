package grpcfx

import (
	"context"
	"log/slog"
	"time"

	"github.com/eser/ajan/logfx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(logger *logfx.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startTime := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(startTime)

		// Log in same format as httpfx
		logger.InfoContext(ctx, "RPC Call",
			slog.String("method", info.FullMethod),
			slog.String("duration", duration.String()),
		)

		return resp, err
	}
}

func MetricsInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startTime := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		st, _ := status.FromError(err)

		// Record metrics with attributes
		attrs := metric.WithAttributes(
			attribute.String("method", info.FullMethod),
			attribute.String("code", st.Code().String()),
		)

		metrics.RequestsTotal.Add(ctx, 1, attrs)

		durationAttrs := metric.WithAttributes(
			attribute.String("method", info.FullMethod),
		)
		metrics.RequestDuration.Record(ctx, duration.Seconds(), durationAttrs)

		return resp, err
	}
}
