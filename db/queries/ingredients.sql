-- name: CreateIngredient :one
INSERT INTO ingredients (id, name, protein, fat, carbohydrates, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    $1, -- name
    $2, -- protein
    $3, -- fat
    $4, -- carbohydrates
    NOW(),
    NOW()
)
RETURNING id, name, protein, fat, carbohydrates, created_at, updated_at;

-- name: GetIngredientById :one
SELECT id, name, protein, fat, carbohydrates, created_at, updated_at
FROM ingredients
WHERE id = $1;

-- name: ListIngredients :many
SELECT id, name, protein, fat, carbohydrates, created_at, updated_at
FROM ingredients
WHERE name ILIKE '%' || sqlc.arg('search_query')::text || '%'
ORDER BY name ASC
LIMIT $1 OFFSET $2;

-- name: UpdateIngredient :one
UPDATE ingredients
SET name = $2, protein = $3, fat = $4, carbohydrates = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, name, protein, fat, carbohydrates, created_at, updated_at;

-- name: DeleteIngredient :exec
DELETE FROM ingredients
WHERE id = $1;