-- name: ListTransactionsForMonth :many
SELECT t.*, c.name AS category_name, c.color AS category_color
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.occurred_on >= $2 AND t.occurred_on < $3
ORDER BY t.occurred_on DESC, t.id DESC;

-- name: CreateTransaction :one
INSERT INTO transactions (user_id, category_id, amount, type, description, occurred_on)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = $1 AND user_id = $2;

-- name: UpdateTransaction :one
UPDATE transactions
SET category_id = $3, amount = $4, type = $5, description = $6, occurred_on = $7, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteTransaction :execrows
DELETE FROM transactions WHERE id = $1 AND user_id = $2;
