-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (token, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPasswordResetToken :one
SELECT * FROM password_reset_tokens WHERE token = $1;

-- name: DeletePasswordResetToken :exec
DELETE FROM password_reset_tokens WHERE token = $1;

-- DeletePasswordResetTokensForUser drops every outstanding reset token for a
-- user, so requesting a new link invalidates any earlier one that might have
-- been forwarded or left sitting in an inbox.
-- name: DeletePasswordResetTokensForUser :exec
DELETE FROM password_reset_tokens WHERE user_id = $1;
