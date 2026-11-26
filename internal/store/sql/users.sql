-- name: organization_upsert
INSERT INTO organization (organization_id)
VALUES ($1)
ON CONFLICT (organization_id) DO NOTHING;

-- name: user_upsert
INSERT INTO user (internal_id, name, organization_id)
VALUES ($1, $2, $3)
ON CONFLICT (internal_id) DO UPDATE
SET name            = EXCLUDED.name,
    organization_id = EXCLUDED.organization_id,
    updated_at      = now()
RETURNING internal_id, name, organization_id, created_at, updated_at;

-- name: user_get
SELECT internal_id, name, organization_id, created_at, updated_at
FROM user
WHERE internal_id = $1;

-- name: user_list_by_organization
SELECT internal_id, name, organization_id, created_at, updated_at
FROM user
WHERE organization_id = $1
ORDER BY created_at, internal_id;

-- name: user_exists
SELECT EXISTS (
    SELECT 1
    FROM user
    WHERE internal_id = $1
);

-- name: user_member_list
SELECT internal_id, role_id, assigned_at
FROM user_member
WHERE internal_id = $1
ORDER BY assigned_at, role_id;

-- name: user_member_delete
DELETE FROM user_member
WHERE internal_id = $1 AND role_id = $2;

-- name: user_member_create
INSERT INTO user_member (internal_id, role_id)
VALUES ($1, $2)
RETURNING internal_id, role_id, assigned_at;