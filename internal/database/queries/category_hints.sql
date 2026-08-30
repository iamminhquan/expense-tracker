-- GetCategoryHint looks up the remembered category for one user's note_key.
-- pgx.ErrNoRows on a miss is not an error condition here -- it is what
-- tells the processing loop to fall back to Other/Other income (and, once
-- a later slice adds it, to ask the classifier).
-- name: GetCategoryHint :one
SELECT * FROM category_hints
WHERE user_id = $1 AND note_key = $2;

-- UpsertCategoryHint is how the memory learns: the processing loop writes
-- one the first time a note_key is seen with no hint yet, and a user
-- correcting the category on an email-sourced transaction writes one that
-- always overwrites whatever was there. ON CONFLICT on (user_id, note_key)
-- is what makes the second case an overwrite instead of a second row.
-- name: UpsertCategoryHint :one
INSERT INTO category_hints (user_id, note_key, category_id)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, note_key) DO UPDATE
SET category_id = $3, updated_at = now()
RETURNING *;

-- name: DeleteCategoryHintsForUser :exec
DELETE FROM category_hints WHERE user_id = $1;
