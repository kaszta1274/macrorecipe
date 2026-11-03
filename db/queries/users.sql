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

-- name: DeleteUsers :exec
DELETE FROM users;