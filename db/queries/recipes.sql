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

-- name: ListRecipesByName :many
SELECT id, user_id, name, description, instructions, portions, created_at, updated_at
FROM recipes
WHERE name ILIKE '%' || sqlc.arg('search_query')::text || '%'
ORDER BY name ASC
LIMIT $1 OFFSET $2;

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