-- name: CreateLike :one
INSERT INTO likes (user_id, recipe_id, created_at)
VALUES (
    $1, -- user_id
    $2, -- recipe_id
    NOW()
)
RETURNING user_id, recipe_id, created_at;

-- name: GetLikeById :one
SELECT user_id, recipe_id, created_at
FROM likes
WHERE user_id = $1 AND recipe_id = $2;

-- name: ListLikesByUserId :many
SELECT 
    likes.user_id,
    likes.recipe_id,
    recipes.name AS recipe_name,
    recipes.description AS recipe_description,
    likes.created_at
FROM likes
JOIN recipes ON likes.recipe_id = recipes.id
WHERE likes.user_id = $1
ORDER BY likes.created_at DESC, likes.recipe_id ASC
LIMIT $2 OFFSET $3;

-- name: CountLikesByRecipeId :one
SELECT COUNT(*) as number_of_likes
FROM likes
WHERE recipe_id = $1;

-- name: DeleteLike :exec
DELETE FROM likes
WHERE user_id = $1 AND recipe_id = $2;