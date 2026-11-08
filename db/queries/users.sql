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
RETURNING id, email, username, role, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, username, password_hash, role, created_at, updated_at 
FROM users
WHERE email = $1;

-- name: GetUserById :one
SELECT id, email, username, password_hash, role, created_at, updated_at 
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET username = $1, password_hash = $2, updated_at = NOW()
RETURNING id, email, username, role, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;