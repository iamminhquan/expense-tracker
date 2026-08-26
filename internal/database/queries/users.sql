-- name: CreateUser :one
INSERT INTO users (email, password_hash, name, username)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserTheme :exec
UPDATE users SET theme = $2 WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: UpdateUserProfile :exec
UPDATE users SET name = $2, username = $3 WHERE id = $1;

-- name: UpdateUserEmail :exec
UPDATE users SET email = $2 WHERE id = $1;

-- RecordFailedLogin counts one wrong password and, on the attempt that
-- reaches max_attempts, stamps the lock. Counting and locking happen in the
-- one statement so two simultaneous guesses can't both read the same count
-- and each write back the same +1 -- a read-modify-write in Go would let a
-- parallel flood spend far more than max_attempts guesses.
-- name: RecordFailedLogin :one
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE
        WHEN failed_login_attempts + 1 >= sqlc.arg(max_attempts)::int THEN sqlc.arg(locked_until)::timestamptz
        ELSE locked_until
    END
WHERE id = sqlc.arg(id)
RETURNING failed_login_attempts, locked_until;

-- ClearFailedLogins wipes the throttle state, called both after a correct
-- password and after a password reset -- the reset is what lets a locked-out
-- owner back in without waiting the window out.
-- name: ClearFailedLogins :exec
UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = $1;
