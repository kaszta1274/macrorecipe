-- name: CreateUser :one
INSERT INTO users (id, email, username, password_hash, role, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    $1, -- email
    $2, -- username
    $3, -- password_hash
    $4, -- role
    NOW(),
    NOW()
)
RETURNING id, email, username, role, avatar_url, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, username, password_hash, role, avatar_url, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserById :one
SELECT id, email, username, password_hash, role, avatar_url, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserPublicProfileById :one
SELECT
    u.id,
    u.username,
    u.avatar_url,
    u.created_at,
    (SELECT COUNT(*) FROM recipes r WHERE r.user_id = u.id)::bigint AS recipes_count,
    (SELECT COUNT(*) FROM follows f WHERE f.following_id = u.id)::bigint AS followers_count,
    (SELECT COUNT(*) FROM follows f WHERE f.follower_id = u.id)::bigint AS following_count,
    EXISTS (
        SELECT 1 FROM follows vf
        WHERE vf.follower_id = sqlc.arg('viewer_id')::uuid AND vf.following_id = u.id
    ) AS followed_by_me
FROM users u
WHERE u.id = sqlc.arg('id')::uuid;

-- name: UpdateUser :one
UPDATE users
SET username = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, email, username, role, avatar_url, created_at, updated_at;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, email, username, role, avatar_url, created_at, updated_at;

-- name: UpdateUserAvatar :one
UPDATE users
SET avatar_url = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, email, username, role, avatar_url, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
