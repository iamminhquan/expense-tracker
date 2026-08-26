-- name: CreateSession :one
INSERT INTO sessions (id, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- DeleteOtherSessionsForUser drops every session of a user except the one
-- making the request, so changing a password signs out the other devices
-- without logging out the person doing the changing.
-- name: DeleteOtherSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1 AND id <> $2;

-- DeleteSessionsForUser drops every session of a user, including the one
-- making the request. A password reset has no current session to spare --
-- the visitor arrived signed out -- so every device gets logged out and the
-- freshly reset password is what signs them back in.
-- name: DeleteSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1;
