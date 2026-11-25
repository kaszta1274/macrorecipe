-- name: CreateRecipe :one
INSERT INTO recipes (id, user_id, name, description, instructions, portions, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    $1, -- user_id
    $2, -- name
    $3, -- description
    $4, -- instructions
    $5, -- portions
    NOW(),
    NOW()
)
RETURNING id, user_id, name, description, instructions, portions, created_at, updated_at;

-- name: GetRecipeById :one
SELECT id, user_id, name, description, instructions, portions, created_at, updated_at
FROM recipes
WHERE id = $1;

-- name: GetRecipeDetailById :one
SELECT
    r.id,
    r.name,
    r.description,
    r.instructions,
    r.portions,
    r.created_at,
    r.updated_at,
    u.id AS author_id,
    u.username AS author_username,
    COALESCE(l.likes_count, 0)::bigint AS likes_count,
    EXISTS (
        SELECT 1 FROM likes ul
        WHERE ul.recipe_id = r.id AND ul.user_id = sqlc.arg('viewer_id')::uuid
    ) AS liked_by_me,
    COALESCE(c.comments_count, 0)::bigint AS comments_count
FROM recipes r
JOIN users u ON u.id = r.user_id
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS likes_count FROM likes WHERE recipe_id = r.id
) l ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS comments_count FROM comments WHERE recipe_id = r.id
) c ON true
WHERE r.id = sqlc.arg('id')::uuid;

-- name: ListRecipeSummaries :many
WITH page AS (
    SELECT r.id, r.user_id, r.name, r.description, r.portions, r.created_at, r.updated_at
    FROM recipes r
    WHERE (sqlc.narg('author_id')::uuid IS NULL OR r.user_id = sqlc.narg('author_id')::uuid)
      AND r.name ILIKE '%' || sqlc.arg('search_query')::text || '%'
      AND (
          cardinality(sqlc.arg('ingredient_ids')::uuid[]) = 0
          OR r.id IN (
              SELECT ri.recipe_id
              FROM recipe_ingredients ri
              WHERE ri.ingredient_id = ANY(sqlc.arg('ingredient_ids')::uuid[])
              GROUP BY ri.recipe_id
              HAVING COUNT(DISTINCT ri.ingredient_id) = cardinality(sqlc.arg('ingredient_ids')::uuid[])
          )
      )
    ORDER BY r.name ASC, r.id ASC
    LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset')
)
SELECT
    p.id,
    p.name,
    p.description,
    p.portions,
    p.created_at,
    p.updated_at,
    u.id AS author_id,
    u.username AS author_username,
    COALESCE(m.protein, 0)::real AS total_protein,
    COALESCE(m.fat, 0)::real AS total_fat,
    COALESCE(m.carbs, 0)::real AS total_carbs,
    COALESCE(l.likes_count, 0)::bigint AS likes_count,
    EXISTS (
        SELECT 1 FROM likes ul
        WHERE ul.recipe_id = p.id AND ul.user_id = sqlc.arg('viewer_id')::uuid
    ) AS liked_by_me,
    COALESCE(c.comments_count, 0)::bigint AS comments_count
FROM page p
JOIN users u ON u.id = p.user_id
LEFT JOIN LATERAL (
    SELECT
        SUM(i.protein * ri.amount / 100.0) AS protein,
        SUM(i.fat * ri.amount / 100.0) AS fat,
        SUM(i.carbohydrates * ri.amount / 100.0) AS carbs
    FROM recipe_ingredients ri
    JOIN ingredients i ON i.id = ri.ingredient_id
    WHERE ri.recipe_id = p.id
) m ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS likes_count FROM likes WHERE recipe_id = p.id
) l ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS comments_count FROM comments WHERE recipe_id = p.id
) c ON true
ORDER BY p.name ASC, p.id ASC;

-- name: ListRecipesByUserId :many
SELECT id, user_id, name, description, instructions, portions, created_at, updated_at
FROM recipes
WHERE user_id = $1 AND name ILIKE '%' || sqlc.arg('search_query')::text || '%'
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: UpdateRecipe :one
UPDATE recipes
SET name = $2, description = $3, instructions = $4, portions = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, name, description, instructions, portions, created_at, updated_at;

-- name: DeleteRecipe :exec
DELETE FROM recipes
WHERE id = $1;