package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const defaultGRPCPort = 50053

type Config struct {
	DatabaseURL string
	GRPCPort    int
	LogLevel    slog.Level
}

func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	port := defaultGRPCPort
	if rawPort := strings.TrimSpace(os.Getenv("GRPC_PORT")); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed < 1 || parsed > 65535 {
			return Config{}, fmt.Errorf("GRPC_PORT must be a valid TCP port")
		}
		port = parsed
	}

	level, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL: databaseURL,
		GRPCPort:    port,
		LogLevel:    level,
	}, nil
}

func (c Config) String() string {
	return fmt.Sprintf(
		"Config{DatabaseURL:<redacted>, GRPCPort:%d, LogLevel:%s}",
		c.GRPCPort,
		c.LogLevel,
	)
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}