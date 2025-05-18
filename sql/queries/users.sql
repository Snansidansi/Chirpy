-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at, email;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUser :one
SELECT *
FROM users
WHERE email = $1;
