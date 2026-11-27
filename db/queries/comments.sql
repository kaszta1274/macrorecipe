-- name: CreateComment :one
INSERT INTO comments (id, user_id, recipe_id, body, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    $1, -- user_id
    $2, -- recipe_id
    $3, -- body
    NOW(),
    NOW()
)
RETURNING id, user_id, recipe_id, body, created_at, updated_at;

-- name: GetCommentById :one
SELECT id, user_id, recipe_id, body, created_at, updated_at
FROM comments
WHERE id = $1;

-- name: ListCommentsByRecipeId :many
SELECT 
    comments.id, 
    comments.user_id, 
    users.username,
    users.avatar_url,
    comments.recipe_id, 
    comments.body, 
    comments.created_at, 
    comments.updated_at
FROM comments
JOIN users ON comments.user_id = users.id
WHERE comments.recipe_id = $1
ORDER BY comments.created_at DESC, comments.id ASC
LIMIT $2 OFFSET $3;

-- name: UpdateComment :one
UPDATE comments
SET body = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, recipe_id, body, created_at, updated_at;

-- name: DeleteComment :exec
DELETE FROM comments
WHERE id = $1;