-- name: GetAdminByPhone :one
SELECT * FROM admins WHERE phone_number = $1;
-- name: GetAdminByID :one
SELECT * FROM admins WHERE id = $1;
-- name: CreateAdmin :one
INSERT INTO admins (first_name, last_name, email, phone_number, is_male, password_hash, totp_secret)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;
-- name: UpdateAdminPassword :exec
UPDATE admins SET password_hash = $2 WHERE id = $1;
-- name: SetAdminTOTPSecret :exec
UPDATE admins SET totp_secret = $2 WHERE id = $1;
-- name: SetAdminAvatar :exec
UPDATE admins SET avatar = $2, avatar_type = $3 WHERE id = $1;
-- name: ClearAdminAvatar :exec
UPDATE admins SET avatar = NULL, avatar_type = '' WHERE id = $1;
-- name: GetAdminAvatar :one
SELECT avatar, avatar_type FROM admins WHERE id = $1;
-- name: UpdateAdminProfile :one
UPDATE admins SET first_name = $2, last_name = $3, email = $4, phone_number = $5, is_male = $6
WHERE id = $1 RETURNING *;
-- name: InsertRefreshToken :one
INSERT INTO admin_refresh_tokens (admin_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING *;
-- name: GetRefreshToken :one
SELECT * FROM admin_refresh_tokens WHERE token_hash = $1;
-- name: RevokeRefreshToken :exec
UPDATE admin_refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL;
-- name: RevokeAllAdminRefreshTokens :exec
UPDATE admin_refresh_tokens SET revoked_at = now() WHERE admin_id = $1 AND revoked_at IS NULL;
-- name: InsertPasswordReset :one
INSERT INTO admin_password_resets (admin_id, code_hash, expires_at) VALUES ($1, $2, $3) RETURNING *;
-- name: InvalidateAdminPasswordResets :exec
UPDATE admin_password_resets SET consumed_at = now() WHERE admin_id = $1 AND consumed_at IS NULL;
-- name: GetLatestPasswordReset :one
SELECT * FROM admin_password_resets WHERE admin_id = $1 AND consumed_at IS NULL AND expires_at > now() ORDER BY created_at DESC LIMIT 1;
-- name: ConsumePasswordReset :exec
UPDATE admin_password_resets SET consumed_at = now() WHERE id = $1;
