package grpcfx

import (
	"context"
	"log/slog"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/metricsfx"
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

		// Use the new gRPC-specific attribute helpers
		attrs := metricsfx.GRPCAttrs(info.FullMethod, st.Code().String())
		methodAttrs := metricsfx.GRPCMethodAttrs(info.FullMethod)

		// Clean, simple metric recording
		metrics.RequestsTotal.Inc(ctx, attrs...)
		metrics.RequestDuration.RecordDuration(ctx, duration, methodAttrs...)

		return resp, err
	}
}
