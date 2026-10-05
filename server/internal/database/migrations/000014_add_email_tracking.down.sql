ALTER TABLE transactions DROP COLUMN bank_email_id;
ALTER TABLE transactions DROP COLUMN source;
DROP INDEX IF EXISTS idx_users_inbox_token;
ALTER TABLE users DROP COLUMN inbox_token;
DROP TABLE bank_emails;
-- Category other_income cố ý ở lại: có thể đã có transaction trỏ vào nó, và
-- transactions.category_id không có ON DELETE clause nào.
