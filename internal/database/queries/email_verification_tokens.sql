-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (token, user_id, email, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetEmailVerificationToken :one
SELECT * FROM email_verification_tokens WHERE token = $1;

-- name: DeleteEmailVerificationToken :exec
DELETE FROM email_verification_tokens WHERE token = $1;

-- DeleteEmailVerificationTokensForUser drops every outstanding verification
-- token for a user, so requesting a new link (a resend, or a second email
-- change before the first was confirmed) invalidates any earlier one.
-- name: DeleteEmailVerificationTokensForUser :exec
DELETE FROM email_verification_tokens WHERE user_id = $1;
