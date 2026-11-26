# API Reference — User Service

## Transport

- Protocol: gRPC over HTTP/2
- Proto package: `user.v1`
- Proto file: `api/proto/user/v1/user.proto`
- Go package: `github.com/JiffyCRM/user/api/user/v1`
- Default port: `50053` (`GRPC_PORT`)
- Reflection: enabled
- Health: standard `grpc.health.v1.Health`, empty service name
- Plaintext transport in v1; TLS/mTLS is not implemented

## Authentication metadata

Every RPC except the health service requires:

```text
x-user-id: <internal_id UUID>
```

The value is the internal user ID from `identity-link`, stored by the UI in
`session.user.id`. A server interceptor validates the UUID. Missing, empty,
duplicate, or malformed values return `Unauthenticated`.

The service trusts the caller network. It does not verify a signature, token,
or claim set in v1.

## Common conventions

All RPCs are unary. All timestamps are `google.protobuf.Timestamp` UTC values.

### Normalization and validation

| Field | Rules |
| --- | --- |
| `organization_id`, `internal_id`, `role_id` | Valid non-nil UUID |
| user `name` | Trimmed, 1–100 chars, no control chars |

Invalid inputs return `InvalidArgument`.

### gRPC status codes

| Code | Meaning |
| --- | --- |
| `OK` | Success |
| `InvalidArgument` | Failed validation or malformed UUID |
| `Unauthenticated` | Missing/invalid `x-user-id` |
| `PermissionDenied` | Caller is not the subject of the request |
| `NotFound` | User or user-member target not found |

## RPC surface

`service UserService {`

- `rpc UpsertUser(UpsertUserRequest) returns (User)`
- `rpc GetUser(GetUserRequest) returns (User)`
- `rpc ListUsersByOrganization(ListUsersByOrganizationRequest) returns (ListUsersResponse)`
- `rpc ListUserMembers(ListUserMembersRequest) returns (ListUserMembersResponse)`
- `rpc CreateUserMember(CreateUserMemberRequest) returns (UserMember)`
- `rpc DeleteUserMember(DeleteUserMemberRequest) returns (google.protobuf.Empty)`

`}`

### UpsertUser

Create or update a user profile. `internal_id` must match the authenticated
caller. Upserts the owning `organization` row idempotently.

### GetUser

Lookup a user profile by `internal_id`. Any authenticated caller may read any
user profile; the service does not enforce organization boundaries on reads in
v1.

### ListUsersByOrganization

List every user profile belonging to an organization.

### ListUserMembers

List the role assignments (`user_member` rows) for a user.

### CreateUserMember

Add a role assignment (`user_member` row) for a user. `internal_id` must match
the authenticated caller — users manage only their own role assignments.

### DeleteUserMember

Remove a single role assignment. `internal_id` must match the authenticated
caller.

## Messages

### User

| Field | Type | Notes |
| --- | --- | --- |
| `internal_id` | string | Opaque UUID from identity-link |
| `name` | string | Profile display name |
| `organization_id` | string | Owning organization UUID |
| `created_at` | Timestamp | |
| `updated_at` | Timestamp | |

### UserMember

| Field | Type | Notes |
| --- | --- | --- |
| `internal_id` | string | User UUID |
| `role_id` | string | Role UUID |
| `assigned_at` | Timestamp | Assignment time |

## Data ownership

- Table `user`: `internal_id` (PK), `name`, `organization_id` (FK →
  `organization`), timestamps.
- Table `user_member`: `internal_id` (FK → `user`), `role_id`,
  `assigned_at`, PK (`internal_id`, `role_id`).
- Table `organization`: `organization_id` (PK), timestamps.
- This service stores only `internal_id` + `name`. It never stores plaintext
  email, provider subject, or other PII.
- Emails and authentication remain in `identity-link`; this service only knows
  the opaque `internal_id`.
