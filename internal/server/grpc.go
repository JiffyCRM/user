package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	userv1 "github.com/JiffyCRM/user/api/user/v1"
	"github.com/JiffyCRM/user/internal/auth"
)

func NewGRPCServer(userService userv1.UserServiceServer, logger *slog.Logger) (*grpc.Server, *health.Server) {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(logger),
			loggingInterceptor(logger),
			auth.UnaryServerInterceptor(),
		),
	)

	userv1.RegisterUserServiceServer(grpcServer, userService)
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	return grpcServer, healthServer
}

func RunHealthLoop(ctx context.Context, pool *pgxpool.Pool, healthServer *health.Server, logger *slog.Logger, interval time.Duration) {
	check := func() {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := pool.Ping(pingCtx); err != nil {
			healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
			logger.Warn("database health check failed", "error", err)
			return
		}
		healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	}

	check()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			healthServer.Shutdown()
			return
		case <-ticker.C:
			check()
		}
	}
}

func recoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic in gRPC handler", "method", info.FullMethod, "panic", recovered)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

func loggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		logger.Info("gRPC request",
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return resp, err
	}
}