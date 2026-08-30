-- name: AdminListBusinesses :many
SELECT b.id, b.name, b.owner_user_id, b.created_at, b.deleted_at, u.first_name AS owner_first_name, u.last_name AS owner_last_name, u.phone_number AS owner_phone_number, (SELECT count(*) FROM business_members WHERE business_id = b.id)::bigint AS member_count FROM businesses b JOIN users u ON u.id = b.owner_user_id WHERE b.deleted_at IS NULL AND (sqlc.narg('name')::text IS NULL OR b.name ILIKE '%' || sqlc.narg('name') || '%') ORDER BY b.created_at DESC LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');
-- name: AdminCountBusinesses :one
SELECT count(*) FROM businesses b WHERE b.deleted_at IS NULL AND (sqlc.narg('name')::text IS NULL OR b.name ILIKE '%' || sqlc.narg('name') || '%');
-- name: AdminGetBusiness :one
SELECT b.id, b.name, b.owner_user_id, b.created_at, b.deleted_at, u.first_name AS owner_first_name, u.last_name AS owner_last_name, u.phone_number AS owner_phone_number FROM businesses b JOIN users u ON u.id = b.owner_user_id WHERE b.id = $1 AND b.deleted_at IS NULL;
-- name: AdminListOwnedBusinesses :many
SELECT b.id, b.name, b.created_at, (SELECT count(*) FROM business_members WHERE business_id = b.id)::bigint AS member_count FROM businesses b WHERE b.owner_user_id = $1 AND b.deleted_at IS NULL ORDER BY b.created_at DESC;
-- name: AdminListJoinedBusinesses :many
SELECT b.id, b.name, m.role, trim(concat_ws(' ', u.first_name, u.last_name)) AS owner_name, b.created_at FROM businesses b JOIN business_members m ON m.business_id = b.id AND m.user_id = $1 JOIN users u ON u.id = b.owner_user_id WHERE m.role <> 'owner' AND b.deleted_at IS NULL ORDER BY b.created_at DESC;
