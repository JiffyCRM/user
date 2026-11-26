# user — JiffyCRM user microservice

Go + gRPC + pgx service that owns user profiles (`internal_id` + `name`),
the organizations users belong to, and per-user role assignments
(`user_member`).

## Data ownership

- Table `user`: `internal_id` (PK), `name`, `organization_id` (FK → `organization`), timestamps.
- Table `user_member`: `internal_id` (FK → `user`), `role_id`, `assigned_at`, PK (`internal_id`, `role_id`).
- Table `organization`: `organization_id` (PK), timestamps.
- Stores only opaque `internal_id` from `identity-link` + profile `name`; no plaintext email or PII.

## Transport

- Protocol: gRPC over HTTP/2
- Proto package: `user.v1`
- Proto file: `api/proto/user/v1/user.proto`
- Go package: `github.com/JiffyCRM/user/api/user/v1`
- Default port: `50053` (`GRPC_PORT`)
- Reflection: enabled
- Health: standard `grpc.health.v1.Health`

## RPC surface

| RPC | Description |
| --- | --- |
| `UpsertUser` | Create/update a user profile; `internal_id` must match the caller |
| `GetUser` | Lookup a user profile by `internal_id` |
| `ListUsersByOrganization` | List every user profile in an organization |
| `ListUserMembers` | List role assignments (`user_member`) for a user |
| `CreateUserMember` | Add a role assignment; `internal_id` must match the caller |
| `DeleteUserMember` | Remove a single role assignment; `internal_id` must match the caller |

## Auth

Every RPC except health requires `x-user-id` gRPC metadata (the opaque
internal user ID from `identity-link`).

## Run

```powershell
# local postgres
docker-compose up -d postgres

# migrations
$env:DATABASE_URL="postgres://user:user@localhost:5435/user?sslmode=disable"
go run ./cmd/migrate up

# server
go run ./cmd/server
```

## Test

```powershell
go test ./internal/service/... -count=1 -v   # unit (no Docker)
# integration needs Docker or TEST_DATABASE_URL:
go test ./internal/store/... -count=1 -v
```

## Commands

| Target | Purpose |
| --- | --- |
| `make generate` | Regenerate proto stubs |
| `make migrate` | Apply DB migrations |
| `make test` | Run all tests |
| `make run` | Start the gRPC server |
| `make build` | Build binary |
| `make docker-build` | Build container image |

## Related services

- `identity-link` — source of `internal_id` (email → ID resolution)
- `membership` — organizations, roles, permissions, invitations
- `overview` — shared Compose stack and service catalog
