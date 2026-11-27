-- name: CreateFollow :one
INSERT INTO follows (follower_id, following_id, created_at)
VALUES (
    $1, -- follower_id
    $2, -- following_id
    NOW()
)
RETURNING follower_id, following_id, created_at;

-- name: GetFollowById :one
SELECT follower_id, following_id, created_at
FROM follows
WHERE follower_id = $1 AND following_id = $2;

-- name: ListFollowersByUserId :many
SELECT 
    u.id,
    u.username,
    u.avatar_url,
    f.created_at AS followed_at,
    EXISTS (
        SELECT 1 FROM follows vf
        WHERE vf.follower_id = sqlc.arg('viewer_id')::uuid AND vf.following_id = u.id
    ) AS followed_by_me
FROM follows f
JOIN users u ON u.id = f.follower_id
WHERE f.following_id = sqlc.arg('user_id')::uuid
ORDER BY f.created_at DESC, u.id ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListFollowingsByUserId :many
SELECT 
    u.id,
    u.username,
    u.avatar_url,
    f.created_at AS followed_at,
    EXISTS (
        SELECT 1 FROM follows vf
        WHERE vf.follower_id = sqlc.arg('viewer_id')::uuid AND vf.following_id = u.id
    ) AS followed_by_me
FROM follows f
JOIN users u ON u.id = f.following_id
WHERE f.follower_id = sqlc.arg('user_id')::uuid
ORDER BY f.created_at DESC, u.id ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DeleteFollow :exec
DELETE FROM follows
WHERE follower_id = $1 AND following_id = $2;