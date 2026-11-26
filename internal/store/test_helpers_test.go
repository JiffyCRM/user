package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/JiffyCRM/user/internal/migrate"
	"github.com/JiffyCRM/user/internal/store"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var dsn string

	if envDSN := os.Getenv("TEST_DATABASE_URL"); envDSN != "" {
		dsn = envDSN
	} else {
		container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
			tcpostgres.WithDatabase("user"),
			tcpostgres.WithUsername("user"),
			tcpostgres.WithPassword("user"),
		)
		if err != nil {
			t.Skipf("Docker is not available for integration tests: %v", err)
		}
		t.Cleanup(func() {
			_ = container.Terminate(context.Background())
		})

		dsn, err = container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("get container connection string: %v", err)
		}
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse database config: %v", err)
	}
	poolConfig.MaxConns = 25

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create database pool: %v", err)
	}
	t.Cleanup(pool.Close)

	deadline := time.Now().Add(45 * time.Second)
	for {
		pingCtx, pingCancel := context.WithTimeout(ctx, 2*time.Second)
		err = pool.Ping(pingCtx)
		pingCancel()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("database did not become ready: %v", err)
		}
		time.Sleep(250 * time.Millisecond)
	}

	if _, err := migrate.RunUp(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return pool
}

func truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		TRUNCATE user_member, user, organization CASCADE
	`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func insertOrganization(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) {
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization (organization_id) VALUES ($1) ON CONFLICT (organization_id) DO NOTHING
	`, orgID); err != nil {
		panic(err)
	}
}
