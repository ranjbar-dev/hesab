ALTER TABLE users
    ADD COLUMN national_id  TEXT,
    ADD COLUMN account_type TEXT NOT NULL DEFAULT 'individual',
    ADD COLUMN status       TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN deleted_at   TIMESTAMPTZ;

ALTER TABLE users DROP CONSTRAINT users_email_key;
ALTER TABLE users ALTER COLUMN email SET DEFAULT '';

CREATE INDEX users_created_at_idx ON users (created_at DESC);
CREATE INDEX users_status_idx     ON users (status) WHERE deleted_at IS NULL;
