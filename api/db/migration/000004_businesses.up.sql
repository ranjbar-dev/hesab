CREATE TABLE businesses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX businesses_owner_idx ON businesses (owner_user_id) WHERE deleted_at IS NULL;
CREATE INDEX businesses_created_at_idx ON businesses (created_at DESC);

CREATE TABLE business_members (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, user_id)
);
CREATE INDEX business_members_user_idx ON business_members (user_id);

CREATE TABLE business_invites (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    invited_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX business_invites_pending_uniq ON business_invites (business_id, user_id) WHERE status = 'pending';
