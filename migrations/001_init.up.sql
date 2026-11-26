CREATE TABLE organization (
    organization_id UUID PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user (
    internal_id     UUID PRIMARY KEY,
    name            TEXT NOT NULL,
    organization_id UUID NOT NULL REFERENCES organization(organization_id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX user_organization_id_idx ON user (organization_id);

CREATE TABLE user_member (
    internal_id UUID NOT NULL REFERENCES user(internal_id) ON DELETE CASCADE,
    role_id     UUID NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (internal_id, role_id)
);

CREATE INDEX user_member_role_id_idx ON user_member (role_id);