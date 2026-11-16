-- name: CreateRecipeIngredient :one
INSERT INTO recipe_ingredients (recipe_id, ingredient_id, amount, created_at, updated_at)
VALUES (
    $1, -- recipe_id
    $2, -- ingredient_id
    $3, -- amount
    NOW(),
    NOW()
)
RETURNING recipe_id, ingredient_id, amount, created_at, updated_at;

-- name: GetRecipeIngredientById :one
SELECT recipe_id, ingredient_id, amount, created_at, updated_at
FROM recipe_ingredients
WHERE recipe_id = $1 AND ingredient_id = $2;

-- name: ListRecipeIngredientsByRecipeId :many
SELECT 
    recipe_ingredients.recipe_id,
    recipe_ingredients.ingredient_id,
    ingredients.name AS ingredient_name, 
    recipe_ingredients.amount, 
    ingredients.protein,
    ingredients.fat,
    ingredients.carbohydrates,
    recipe_ingredients.created_at, 
    recipe_ingredients.updated_at
FROM recipe_ingredients
JOIN ingredients ON recipe_ingredients.ingredient_id = ingredients.id
WHERE recipe_ingredients.recipe_id = $1
ORDER BY ingredients.name ASC;

-- name: UpdateRecipeIngredient :one
UPDATE recipe_ingredients
SET amount = $3, updated_at = NOW()
WHERE recipe_id = $1 AND ingredient_id = $2
RETURNING recipe_id, ingredient_id, amount, created_at, updated_at;

-- name: DeleteRecipeIngredient :exec
DELETE FROM recipe_ingredients
WHERE recipe_id = $1 AND ingredient_id = $2;