-- name: CreateUser :one
INSERT INTO users (email, password_hash, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserTheme :exec
UPDATE users SET theme = $2 WHERE id = $1;
