-- name: ListCategoriesForUser :many
SELECT * FROM categories
WHERE user_id = $1 OR user_id IS NULL
ORDER BY user_id NULLS FIRST, name;

-- name: GetCategoryForUser :one
SELECT * FROM categories
WHERE id = $1 AND (user_id = $2 OR user_id IS NULL);

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, type, color)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteCategory :execrows
DELETE FROM categories WHERE id = $1 AND user_id = $2;

-- name: UpdateCategoryName :one
UPDATE categories SET name = $3
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateCategoryColor :one
UPDATE categories SET color = $3
WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
RETURNING *;

-- name: GetDefaultCategoryForReassignment :one
SELECT * FROM categories
WHERE user_id IS NULL AND type = 'expense' AND name = 'Khác';

-- name: ListCategoriesWithTransactionCounts :many
SELECT c.*, COUNT(t.id) AS transaction_count
FROM categories c
LEFT JOIN transactions t ON t.category_id = c.id AND t.user_id = $1
WHERE c.user_id = $1 OR c.user_id IS NULL
GROUP BY c.id
ORDER BY c.user_id NULLS FIRST, c.name;
