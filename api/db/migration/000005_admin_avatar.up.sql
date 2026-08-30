-- ponytail: avatar stored as bytea on the admins row — there are a handful of super-admins, not thousands of users, so no object storage / volume mount / nginx location is worth it. Revisit if the admins table ever grows large or avatars get bigger than a small profile photo.
ALTER TABLE admins ADD COLUMN avatar BYTEA;
ALTER TABLE admins ADD COLUMN avatar_type TEXT NOT NULL DEFAULT '';
