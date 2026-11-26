-- name: schema_migrations_create
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    BIGINT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- name: migration_lock_acquire
SELECT pg_advisory_lock($1);

-- name: migration_lock_release
SELECT pg_advisory_unlock($1);

-- name: migration_applied_list
SELECT version
FROM schema_migrations;

-- name: migration_latest_version
SELECT COALESCE(MAX(version), 0)
FROM schema_migrations;

-- name: migration_version_record
INSERT INTO schema_migrations (version)
VALUES ($1);

-- name: migration_version_delete
DELETE FROM schema_migrations
WHERE version = $1;