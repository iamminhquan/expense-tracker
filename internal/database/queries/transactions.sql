-- ListTransactionsForMonth returns one page of the month's transactions,
-- narrowed by whatever the search box and filter panel are asking for. The
-- list page is paginated (see pageSize in internal/handlers/paging.go), so
-- every caller wants a window rather than the whole month; a caller that only
-- needs how many there are should use CountTransactionsForMonth instead of
-- reading rows it will throw away.
--
-- The five filter parameters are all nullable, and a NULL switches its own
-- predicate off -- which is why the filters live here rather than in a second
-- query: the unfiltered list is just this one with five NULLs, so there is
-- only ever one code path to keep in step with CountTransactionsForMonth.
-- That matters because the count is what the pager and the count chip are
-- built from; a count that did not know about the filters would promise
-- pages the list cannot fill.
--
-- The search is a case-insensitive substring match, and it covers the category
-- as well as the note, because both are on screen in every row: someone who
-- can see "Transport" in the list will type it into the search box.
--
-- Which is why the category arrives as two predicates rather than one. A
-- category the user created shows the name they typed, so the name column is
-- what to match -- but only for those, which is what the NULL slug tests. A
-- default category shows a label that lives in internal/i18n, keyed by slug,
-- and its name column is only ever a fallback; matching it would search
-- whatever a migration left there instead of what the row displays. So the
-- handler translates the term into the slugs whose labels contain it and
-- passes those down, keeping the match on the one thing that identifies a
-- default category.
--
-- None of this can use an index and none of it needs to: the month window in
-- front of it is already covered by idx_transactions_user_id_occurred_on, and
-- one person's month is not a large scan.
-- name: ListTransactionsForMonth :many
SELECT t.*, c.slug AS category_slug, c.name AS category_name, c.color AS category_color
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.occurred_on >= $2 AND t.occurred_on < $3
  AND (sqlc.narg('search')::text IS NULL
       OR t.description ILIKE '%' || sqlc.narg('search')::text || '%'
       OR (c.slug IS NULL AND c.name ILIKE '%' || sqlc.narg('search')::text || '%')
       OR c.slug = ANY(sqlc.narg('search_slugs')::text[]))
  AND (sqlc.narg('type')::text IS NULL OR t.type = sqlc.narg('type')::text)
  AND (sqlc.narg('category_id')::bigint IS NULL OR t.category_id = sqlc.narg('category_id')::bigint)
  AND (sqlc.narg('min_amount')::bigint IS NULL OR t.amount >= sqlc.narg('min_amount')::bigint)
  AND (sqlc.narg('max_amount')::bigint IS NULL OR t.amount <= sqlc.narg('max_amount')::bigint)
ORDER BY t.occurred_on DESC, t.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- CountTransactionsForMonth answers how many rows the list above would have,
-- and so carries exactly the same filter predicates. Keep the two WHERE
-- clauses identical: everything that reads a page -- the pager's page count,
-- the count chip, the empty state -- trusts that they agree.
-- name: CountTransactionsForMonth :one
SELECT COUNT(*)::bigint AS count
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.user_id = $1 AND t.occurred_on >= $2 AND t.occurred_on < $3
  AND (sqlc.narg('search')::text IS NULL
       OR t.description ILIKE '%' || sqlc.narg('search')::text || '%'
       OR (c.slug IS NULL AND c.name ILIKE '%' || sqlc.narg('search')::text || '%')
       OR c.slug = ANY(sqlc.narg('search_slugs')::text[]))
  AND (sqlc.narg('type')::text IS NULL OR t.type = sqlc.narg('type')::text)
  AND (sqlc.narg('category_id')::bigint IS NULL OR t.category_id = sqlc.narg('category_id')::bigint)
  AND (sqlc.narg('min_amount')::bigint IS NULL OR t.amount >= sqlc.narg('min_amount')::bigint)
  AND (sqlc.narg('max_amount')::bigint IS NULL OR t.amount <= sqlc.narg('max_amount')::bigint);

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
