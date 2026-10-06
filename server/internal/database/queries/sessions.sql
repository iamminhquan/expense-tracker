-- name: CreateSession :one
INSERT INTO sessions (id, user_id, expires_at, user_agent)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- DeleteSession matches the token just replaced as well as the current one,
-- so a logout that races a refresh still ends the session.
-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1 OR previous_id = $1;

-- RotateSession swaps an unexpired session's token for a new one in place, so
-- the row (public_id, created_at, user_agent, expires_at) is the same session
-- throughout. It matches nothing if the token is unknown, expired or already
-- replaced, which is how two refreshes racing on one token are told apart:
-- exactly one wins. The right-hand sides read the row as it was before the
-- update, so previous_id gets the old token.
-- name: RotateSession :one
UPDATE sessions SET previous_id = id, id = $2, rotated_at = now()
WHERE id = $1 AND expires_at > now()
RETURNING *;

-- name: GetSessionByPreviousID :one
SELECT * FROM sessions WHERE previous_id = $1;

-- name: ListSessionsForUser :many
SELECT * FROM sessions WHERE user_id = $1 ORDER BY created_at DESC;

-- DeleteSessionForUser is the scoped counterpart to DeleteSession: it takes
-- a user_id as well as the session's public_id (never its id, which is the
-- refresh token and is not sent to clients) so that an id a client sends
-- can only ever delete a session owned by the caller.
-- name: DeleteSessionForUser :exec
DELETE FROM sessions WHERE public_id = $1 AND user_id = $2;

-- DeleteOtherSessionsForUser drops every session of a user except the one
-- making the request, so changing a password signs out the other devices
-- without logging out the person doing the changing.
-- name: DeleteOtherSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1 AND id <> $2 AND previous_id IS DISTINCT FROM $2;

-- DeleteSessionsForUser drops every session of a user, including the one
-- making the request. A password reset has no current session to spare --
-- the visitor arrived signed out -- so every device gets logged out and the
-- freshly reset password is what signs them back in.
-- name: DeleteSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1;
