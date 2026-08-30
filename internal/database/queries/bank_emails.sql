-- CreateBankEmail lưu một thư vừa tới. ON CONFLICT DO NOTHING vì unique index
-- trên (user_id, message_id) là thứ chặn thư trùng, và một thư trùng không
-- phải lỗi để trả về cho Worker -- Worker không làm gì được với nó.
-- name: CreateBankEmail :one
INSERT INTO bank_emails (user_id, message_id, from_address, subject, body, raw_body, status, failure_reason)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
