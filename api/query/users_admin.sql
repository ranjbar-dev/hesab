-- name: AdminCreateUser :one
INSERT INTO users (first_name, last_name, email, phone_number, national_id, account_type, password_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: AdminGetUser :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: AdminListUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg('first_name')::text IS NULL OR first_name ILIKE '%' || sqlc.narg('first_name') || '%')
  AND (sqlc.narg('last_name')::text IS NULL OR last_name ILIKE '%' || sqlc.narg('last_name') || '%')
  AND (sqlc.narg('phone')::text IS NULL OR phone_number ILIKE '%' || sqlc.narg('phone') || '%')
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at < sqlc.narg('created_to'))
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: AdminCountUsers :one
SELECT count(*) FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg('first_name')::text IS NULL OR first_name ILIKE '%' || sqlc.narg('first_name') || '%')
  AND (sqlc.narg('last_name')::text IS NULL OR last_name ILIKE '%' || sqlc.narg('last_name') || '%')
  AND (sqlc.narg('phone')::text IS NULL OR phone_number ILIKE '%' || sqlc.narg('phone') || '%')
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at < sqlc.narg('created_to'));

-- name: AdminUpdateUserProfile :one
UPDATE users SET first_name = $2, last_name = $3, email = $4, national_id = $5, account_type = $6
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: AdminSetUserStatus :one
UPDATE users SET status = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: AdminSetUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: AdminSoftDeleteUser :exec
UPDATE users SET deleted_at = now(), status = 'disabled' WHERE id = $1 AND deleted_at IS NULL;
