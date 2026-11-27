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
WITH page AS (
    SELECT l.recipe_id, l.created_at AS liked_at
    FROM likes l
    WHERE l.user_id = sqlc.arg('user_id')::uuid
    ORDER BY l.created_at DESC, l.recipe_id ASC
    LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset')
)
SELECT
    r.id,
    r.name,
    r.description,
    r.portions,
    r.image_url,
    r.created_at,
    r.updated_at,
    p.liked_at,
    u.id AS author_id,
    u.username AS author_username,
    u.avatar_url AS author_avatar_url,
    COALESCE(m.protein, 0)::real AS total_protein,
    COALESCE(m.fat, 0)::real AS total_fat,
    COALESCE(m.carbs, 0)::real AS total_carbs,
    COALESCE(lc.likes_count, 0)::bigint AS likes_count,
    EXISTS (
        SELECT 1 FROM likes ul
        WHERE ul.recipe_id = r.id AND ul.user_id = sqlc.arg('viewer_id')::uuid
    ) AS liked_by_me,
    COALESCE(c.comments_count, 0)::bigint AS comments_count
FROM page p
JOIN recipes r ON r.id = p.recipe_id
JOIN users u ON u.id = r.user_id
LEFT JOIN LATERAL (
    SELECT
        SUM(i.protein * ri.amount / 100.0) AS protein,
        SUM(i.fat * ri.amount / 100.0) AS fat,
        SUM(i.carbohydrates * ri.amount / 100.0) AS carbs
    FROM recipe_ingredients ri
    JOIN ingredients i ON i.id = ri.ingredient_id
    WHERE ri.recipe_id = r.id
) m ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS likes_count FROM likes WHERE recipe_id = r.id
) lc ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS comments_count FROM comments WHERE recipe_id = r.id
) c ON true
ORDER BY p.liked_at DESC, r.id ASC;

-- name: CountLikesByRecipeId :one
SELECT COUNT(*) as number_of_likes
FROM likes
WHERE recipe_id = $1;

-- name: DeleteLike :exec
DELETE FROM likes
WHERE user_id = $1 AND recipe_id = $2;