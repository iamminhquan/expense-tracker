-- name: ListTransactionsForMonth :many
SELECT t.*, c.slug AS category_slug, c.name AS category_name, c.color AS category_color
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

-- MonthlyTotals returns the displayed month's own two totals plus the
-- balance carried into it. The carry-in rides along on this query rather
-- than getting one of its own because all four callers that build a balance
-- already run this one, and a separate query would mean a second round trip
-- at each of them.
--
-- The outer WHERE therefore reaches back over the user's whole history
-- rather than fencing the month on both sides, and each column narrows from
-- there: the two month totals re-apply the lower bound in their FILTER,
-- while carried_over takes everything strictly before it. Both predicates
-- are still covered by idx_transactions_user_id_occurred_on.
--
-- The ::bigint on carried_over wraps the whole subtraction rather than each
-- operand: sqlc types a bare binary expression as int32, which would overflow
-- a balance past 2.1 tỷ đồng.
-- name: MonthlyTotals :one
SELECT
    COALESCE(SUM(amount) FILTER (WHERE type = 'expense' AND occurred_on >= $2), 0)::bigint AS total_expense,
    COALESCE(SUM(amount) FILTER (WHERE type = 'income' AND occurred_on >= $2), 0)::bigint AS total_income,
    (COALESCE(SUM(amount) FILTER (WHERE type = 'income' AND occurred_on < $2), 0)
      - COALESCE(SUM(amount) FILTER (WHERE type = 'expense' AND occurred_on < $2), 0))::bigint AS carried_over
FROM transactions
WHERE user_id = $1 AND occurred_on < $3;

-- name: CategoryBreakdown :many
SELECT c.slug AS category_slug, c.name AS category_name, c.color AS category_color, SUM(t.amount)::bigint AS total
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.type = 'expense' AND t.occurred_on >= $2 AND t.occurred_on < $3
GROUP BY c.slug, c.name, c.color
ORDER BY total DESC;

-- name: ReassignCategoryTransactions :execrows
UPDATE transactions SET category_id = $1
WHERE category_id = $2 AND user_id = $3;

-- name: CountTransactionsForCategory :one
SELECT COUNT(*)::bigint AS count FROM transactions
WHERE category_id = $1 AND user_id = $2;

-- name: ListDistinctTransactionMonths :many
SELECT DISTINCT date_trunc('month', occurred_on)::date AS month
FROM transactions
WHERE user_id = $1
ORDER BY month DESC;

-- name: GetTransactionWithCategory :one
SELECT t.*, c.slug AS category_slug, c.name AS category_name, c.color AS category_color
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.id = $1 AND t.user_id = $2;

-- name: MonthlyTotalsSeries :many
SELECT
    date_trunc('month', occurred_on)::date AS month,
    COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)::bigint AS total_expense,
    COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0)::bigint AS total_income
FROM transactions
WHERE user_id = $1 AND occurred_on >= $2 AND occurred_on < $3
GROUP BY month
ORDER BY month;
