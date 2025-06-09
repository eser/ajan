package grpcfx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/eser/ajan/logfx"
	"github.com/eser/ajan/metricsfx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	ErrFailedToCreateGRPCMetrics = errors.New("failed to create gRPC metrics")
	ErrGRPCServiceNetListenError = errors.New("gRPC service net listen error")
)

type GRPCService struct {
	InnerServer  *grpc.Server
	InnerMetrics *Metrics
	Config       *Config
	logger       *logfx.Logger
}

func NewGRPCService(
	config *Config,
	metricsProvider *metricsfx.MetricsProvider,
	logger *logfx.Logger,
) *GRPCService {
	metrics := NewMetrics(metricsProvider)

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			LoggingInterceptor(logger),
			MetricsInterceptor(metrics),
		),
	)

	if config.Reflection {
		reflection.Register(server)
	}

	return &GRPCService{
		InnerServer:  server,
		InnerMetrics: metrics,

		Config: config,
		logger: logger,
	}
}

func (gs *GRPCService) Server() *grpc.Server {
	return gs.InnerServer
}

func (gs *GRPCService) RegisterService(desc *grpc.ServiceDesc, impl any) {
	gs.InnerServer.RegisterService(desc, impl)
}

func (gs *GRPCService) Start(ctx context.Context) (func(), error) {
	gs.logger.InfoContext(ctx, "GRPCService is starting...", slog.String("addr", gs.Config.Addr))

	if gs.InnerMetrics == nil {
		if err := gs.InnerMetrics.Init(); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToCreateGRPCMetrics, err)
		}
	}

	listener, err := net.Listen("tcp", gs.Config.Addr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGRPCServiceNetListenError, err)
	}

	go func() {
		if err := gs.InnerServer.Serve(listener); err != nil {
			gs.logger.ErrorContext(ctx, "GRPCService Serve error", slog.Any("error", err))
		}
	}()

	cleanup := func() {
		gs.logger.InfoContext(ctx, "Shutting down gRPC server...")

		stopped := make(chan struct{})
		go func() {
			gs.InnerServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
			gs.logger.InfoContext(ctx, "GRPCService has gracefully stopped.")
		case <-time.After(gs.Config.GracefulShutdownTimeout):
			gs.logger.WarnContext(ctx, "GRPCService shutdown timeout exceeded, forcing stop")
			gs.InnerServer.Stop()
		}
	}

	return cleanup, nil
}
