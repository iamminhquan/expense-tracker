-- Gỡ bỏ toàn bộ schema của tính năng bank-email auto-tracking (thêm dần qua
-- 000014-000017), vì tính năng đó đã bị xoá hoàn toàn khỏi code: không còn
-- route /inbox/{token}, không còn Cloudflare Email Worker, không còn
-- internal/bankmail, internal/inbound, internal/inboxproc, internal/classify.
--
-- DROP COLUMN không xoá row nào -- amount/category/date/description của
-- mọi transaction vẫn còn nguyên, chỉ mất 2 cột đánh dấu "giao dịch này tới
-- từ email". DROP TABLE xoá vĩnh viễn lịch sử email thô (bank_emails), các
-- category hint đã học (category_hints), và các số tài khoản đã ghi nhận
-- (bank_accounts) -- không có cách nào khôi phục sau khi migration này chạy
-- trên một DB thật, ngoài point-in-time recovery.
--
-- Category 'other_income' mà 000014 seed kèm KHÔNG bị xoá ở đây: nó dùng
-- chung hệ thống category (giống 'other'), không phải riêng của feature
-- này, và có thể đã có transaction thật trỏ vào nó.
ALTER TABLE transactions DROP COLUMN bank_email_id;
ALTER TABLE transactions DROP COLUMN source;
DROP TABLE IF EXISTS bank_accounts;
DROP TABLE IF EXISTS category_hints;
DROP TABLE IF EXISTS bank_emails;
DROP INDEX IF EXISTS idx_users_inbox_token;
ALTER TABLE users DROP COLUMN inbox_token;
