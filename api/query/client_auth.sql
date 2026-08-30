-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone_number = $1;
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
-- name: CreateUser :one
INSERT INTO users (first_name, last_name, email, phone_number, password_hash, totp_secret)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;
-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;
-- name: SetUserTOTPSecret :exec
UPDATE users SET totp_secret = $2 WHERE id = $1;
-- name: InsertUserRefreshToken :one
INSERT INTO user_refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING *;
-- name: GetUserRefreshToken :one
SELECT * FROM user_refresh_tokens WHERE token_hash = $1;
-- name: RevokeUserRefreshToken :exec
UPDATE user_refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL;
-- name: RevokeAllUserRefreshTokens :exec
UPDATE user_refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL;
-- name: InsertUserPasswordReset :one
INSERT INTO user_password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3) RETURNING *;
-- name: InvalidateUserPasswordResets :exec
UPDATE user_password_resets SET consumed_at = now() WHERE user_id = $1 AND consumed_at IS NULL;
-- name: GetLatestUserPasswordReset :one
SELECT * FROM user_password_resets WHERE user_id = $1 AND consumed_at IS NULL AND expires_at > now() ORDER BY created_at DESC LIMIT 1;
-- name: ConsumeUserPasswordReset :exec
UPDATE user_password_resets SET consumed_at = now() WHERE id = $1;
