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

-- SetPendingEmail stages a requested address without touching the one the
-- account still logs in and receives a password-reset link at, so a typo
-- here can never cost the owner their recovery path -- see
-- ApplyVerifiedEmail for the confirm side of this.
-- name: SetPendingEmail :exec
UPDATE users SET pending_email = $2 WHERE id = $1;

-- ApplyVerifiedEmail is what a clicked verification link runs: it proves
-- email is reachable, so it becomes the account's real address, marks the
-- account verified, and clears pending_email regardless of whether this was
-- a signup verification (email already equalled users.email) or a change
-- (email came from pending_email).
-- name: ApplyVerifiedEmail :exec
UPDATE users SET email = $2, email_verified = true, pending_email = NULL WHERE id = $1;

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

-- DeleteUser erases the account itself. Sessions, password-reset tokens and
-- email-verification tokens all reference users ON DELETE CASCADE and go
-- with it; the address is freed for a fresh signup, which is the point --
-- reserving it would mean keeping the one piece of personal data the owner
-- just asked to be rid of.
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- SetInboxToken bật hoặc tắt địa chỉ nhận email của tài khoản. NULL = tắt, và
-- tắt rồi bật lại sinh token khác, nên đây cũng là đường thu hồi.
-- name: SetInboxToken :exec
UPDATE users SET inbox_token = $2 WHERE id = $1;

-- GetUserByInboxToken là lớp xác thực thứ nhất của webhook: token trong địa chỉ
-- phải map ra đúng một tài khoản.
-- name: GetUserByInboxToken :one
SELECT * FROM users WHERE inbox_token = $1;
