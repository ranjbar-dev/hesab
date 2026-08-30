-- name: CreateBusiness :one
INSERT INTO businesses (name, owner_user_id) VALUES ($1, $2) RETURNING *;
-- name: AddMember :one
INSERT INTO business_members (business_id, user_id, role) VALUES ($1, $2, $3) RETURNING *;
-- name: GetBusiness :one
SELECT * FROM businesses WHERE id = $1 AND deleted_at IS NULL;
-- name: GetMemberRole :one
SELECT role FROM business_members WHERE business_id = $1 AND user_id = $2;
-- name: ListUserBusinesses :many
SELECT b.id, b.name, b.owner_user_id, b.created_at, m.role FROM businesses b JOIN business_members m ON m.business_id = b.id AND m.user_id = $1 WHERE b.deleted_at IS NULL ORDER BY b.created_at;
-- name: ListMembers :many
SELECT m.user_id, m.role, u.first_name, u.last_name, u.phone_number, m.created_at FROM business_members m JOIN users u ON u.id = m.user_id WHERE m.business_id = $1 ORDER BY m.created_at;
-- name: RenameBusiness :one
UPDATE businesses SET name = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING *;
-- name: SoftDeleteBusiness :exec
UPDATE businesses SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;
-- name: UpdateMemberRole :one
UPDATE business_members SET role = $3 WHERE business_id = $1 AND user_id = $2 RETURNING *;
-- name: RemoveMember :exec
DELETE FROM business_members WHERE business_id = $1 AND user_id = $2;
-- name: GetActiveUserByPhone :one
SELECT * FROM users WHERE phone_number = $1 AND deleted_at IS NULL;
-- name: CreateInvite :one
INSERT INTO business_invites (business_id, user_id, role, invited_by) VALUES ($1, $2, $3, $4) RETURNING *;
-- name: ListPendingInvitesForUser :many
SELECT i.id, i.business_id, b.name AS business_name, i.role, i.status, COALESCE(NULLIF(trim(concat_ws(' ', u.first_name, u.last_name)), ''), '') AS invited_by_name, i.created_at FROM business_invites i JOIN businesses b ON b.id = i.business_id LEFT JOIN users u ON u.id = i.invited_by WHERE i.user_id = $1 AND i.status = 'pending' AND b.deleted_at IS NULL ORDER BY i.created_at;
-- name: ListPendingInvitesForBusiness :many
SELECT i.id, i.business_id, i.user_id, u.first_name, u.last_name, u.phone_number, i.role, i.created_at FROM business_invites i JOIN users u ON u.id = i.user_id JOIN businesses b ON b.id = i.business_id WHERE i.business_id = $1 AND i.status = 'pending' AND b.deleted_at IS NULL ORDER BY i.created_at;
-- name: GetInvite :one
SELECT * FROM business_invites WHERE id = $1;
-- name: SetInviteStatus :exec
UPDATE business_invites SET status = $2, responded_at = now() WHERE id = $1;
