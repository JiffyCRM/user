package migrate

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JiffyCRM/user/migrations"
)

func RunUp(ctx context.Context, pool *pgxpool.Pool) ([]int64, error) {
	files, err := loadFiles()
	if err != nil {
		return nil, err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire database connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, query("migration_lock_acquire"), advisoryLockKey); err != nil {
		return nil, fmt.Errorf("acquire migration lock: %w", err)
	}
	defer unlock(conn, ctx)

	if _, err := conn.Exec(ctx, query("schema_migrations_create")); err != nil {
		return nil, fmt.Errorf("create schema_migrations table: %w", err)
	}

	appliedVersions, err := loadAppliedVersions(ctx, conn)
	if err != nil {
		return nil, err
	}

	var applied []int64
	for _, version := range files.versions {
		if appliedVersions[version] {
			continue
		}

		body, err := migrations.FS.ReadFile(files.up[version])
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", files.up[version], err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin migration %d: %w", version, err)
		}

		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return nil, fmt.Errorf("apply migration %d: %w", version, err)
		}
		if _, err := tx.Exec(ctx, query("migration_version_record"), version); err != nil {
			_ = tx.Rollback(ctx)
			return nil, fmt.Errorf("record migration %d: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit migration %d: %w", version, err)
		}

		applied = append(applied, version)
	}

	return applied, nil
}

func RunDown(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	files, err := loadFiles()
	if err != nil {
		return 0, err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("acquire database connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, query("migration_lock_acquire"), advisoryLockKey); err != nil {
		return 0, fmt.Errorf("acquire migration lock: %w", err)
	}
	defer unlock(conn, ctx)

	if _, err := conn.Exec(ctx, query("schema_migrations_create")); err != nil {
		return 0, fmt.Errorf("create schema_migrations table: %w", err)
	}

	var latest int64
	err = conn.QueryRow(ctx, query("migration_latest_version")).Scan(&latest)
	if err != nil {
		return 0, fmt.Errorf("find latest migration: %w", err)
	}
	if latest == 0 {
		return 0, nil
	}

	downFile, ok := files.down[latest]
	if !ok {
		return 0, fmt.Errorf("missing down migration for version %d", latest)
	}

	body, err := migrations.FS.ReadFile(downFile)
	if err != nil {
		return 0, fmt.Errorf("read migration %s: %w", downFile, err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin rollback %d: %w", latest, err)
	}
	if _, err := tx.Exec(ctx, string(body)); err != nil {
		_ = tx.Rollback(ctx)
		return 0, fmt.Errorf("roll back migration %d: %w", latest, err)
	}
	if _, err := tx.Exec(ctx, query("migration_version_delete"), latest); err != nil {
		_ = tx.Rollback(ctx)
		return 0, fmt.Errorf("remove migration record %d: %w", latest, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit rollback %d: %w", latest, err)
	}

	return latest, nil
}