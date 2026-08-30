DROP INDEX users_status_idx;
DROP INDEX users_created_at_idx;
ALTER TABLE users ALTER COLUMN email DROP DEFAULT;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
ALTER TABLE users
    DROP COLUMN deleted_at,
    DROP COLUMN status,
    DROP COLUMN account_type,
    DROP COLUMN national_id;
