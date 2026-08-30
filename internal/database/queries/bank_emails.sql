-- CreateBankEmail lưu một thư vừa tới. ON CONFLICT DO NOTHING vì unique index
-- trên (user_id, message_id) là thứ chặn thư trùng, và một thư trùng không
-- phải lỗi để trả về cho Worker -- Worker không làm gì được với nó.
-- name: CreateBankEmail :one
INSERT INTO bank_emails (user_id, message_id, from_address, subject, body, status, failure_reason)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, message_id) DO NOTHING
RETURNING *;

-- name: ListRecentFailedBankEmails :many
SELECT * FROM bank_emails
WHERE user_id = $1 AND status = 'failed'
ORDER BY received_at DESC
LIMIT $2;

-- ListRecentBankEmails is what the settings card actually shows: the
-- owner's most recent forwards regardless of status, so a forwarded email
-- shows up right away as 'pending' rather than only appearing once (or if)
-- it ever reaches 'failed'.
-- name: ListRecentBankEmails :many
SELECT * FROM bank_emails
WHERE user_id = $1
ORDER BY received_at DESC
LIMIT $2;

-- RequeueFailedBankEmails là nút "Thử lại": đặt mọi thư failed về pending để
-- lượt xử lý sau đọc lại chúng bằng parser đã sửa. Đây là toàn bộ lý do chọn
-- phương án lưu email thô.
-- name: RequeueFailedBankEmails :exec
UPDATE bank_emails SET status = 'pending', failure_reason = '', processed_at = NULL
WHERE user_id = $1 AND status = 'failed';

-- name: CountBankEmailsForUser :one
SELECT count(*) FROM bank_emails WHERE user_id = $1;

-- name: DeleteBankEmailsForUser :exec
DELETE FROM bank_emails WHERE user_id = $1;

-- ClaimPendingBankEmail giành một email pending cho đúng một goroutine: chỉ
-- goroutine nào UPDATE trúng dòng còn 'pending' mới nhận được nó về, mọi
-- goroutine khác chạy cùng lúc nhận không hàng nào (pgx.ErrNoRows) và phải bỏ
-- qua email này. Đọc rồi mới ghi sẽ để hai goroutine cùng xử lý một email và
-- tạo hai transaction từ một thư.
-- name: ClaimPendingBankEmail :one
UPDATE bank_emails SET status = 'processing'
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- ListPendingBankEmailIDs is what the processing loop walks: every pending
-- email for the user, not just the one that just arrived, so a Render
-- restart that left one mid-flight gets swept up by whichever email
-- processes next instead of needing a cron job of its own.
-- name: ListPendingBankEmailIDs :many
SELECT id FROM bank_emails
WHERE user_id = $1 AND status = 'pending'
ORDER BY received_at;

-- MarkBankEmailImported closes an email that became a transaction. occurred_at
-- keeps the full Notice.OccurredAt timestamp; transactions.occurred_on only
-- ever holds the date part of it.
-- name: MarkBankEmailImported :exec
UPDATE bank_emails SET status = 'imported', occurred_at = $2, processed_at = now()
WHERE id = $1;

-- MarkBankEmailFailed is the "needs fixing" list: our own error, not the
-- sender's mail. failure_reason is what the retry button on /settings reads
-- back, and processed_at records when the read was found bad.
-- name: MarkBankEmailFailed :exec
UPDATE bank_emails SET status = 'failed', failure_reason = $2, processed_at = now()
WHERE id = $1;

-- MarkBankEmailIgnored covers an unknown sender or a body that parses to no
-- transaction (OTP, ad, a failed transfer). Kept separate from 'failed' so
-- that list stays a to-do rather than filling with ordinary bank mail.
-- name: MarkBankEmailIgnored :exec
UPDATE bank_emails SET status = 'ignored', failure_reason = $2, processed_at = now()
WHERE id = $1;
