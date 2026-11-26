package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JiffyCRM/user/internal/config"
	"github.com/JiffyCRM/user/internal/server"
	"github.com/JiffyCRM/user/internal/service"
	"github.com/JiffyCRM/user/internal/store"
	"github.com/JiffyCRM/user/internal/version"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Println(version.Version)
		return
	}

	if err := run(); err != nil {
		slog.Error("user server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	startupCtx, cancelStartup := context.WithTimeout(ctx, 10*time.Second)
	defer cancelStartup()
	if err := pool.Ping(startupCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	st := store.NewPostgres(pool)
	userService := service.New(st, logger)
	grpcServer, healthServer := server.NewGRPCServer(userService, logger)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("listen on gRPC port %d: %w", cfg.GRPCPort, err)
	}

	healthCtx, cancelHealth := context.WithCancel(ctx)
	defer cancelHealth()
	go server.RunHealthLoop(healthCtx, pool, healthServer, logger, 5*time.Second)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	logger.Info("user gRPC server started",
		"port", cfg.GRPCPort,
		"version", version.Version,
	)

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("serve gRPC: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down user server")
	}

	cancelHealth()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		grpcServer.Stop()
	}

	return nil
}