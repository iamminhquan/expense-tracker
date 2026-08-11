-- name: ListCategoriesForUser :many
SELECT * FROM categories
WHERE user_id = $1 OR user_id IS NULL
ORDER BY user_id NULLS FIRST, name;

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, type, color)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM categories WHERE id = $1 AND user_id = $2;
