-- Khôi phục lại đúng schema của 000014-000017 (gộp lại thành một bước), để
-- revert bằng được. Không khôi phục được DỮ LIỆU đã mất khi .up.sql DROP
-- TABLE/DROP COLUMN -- chỉ tạo lại schema rỗng.
ALTER TABLE users ADD COLUMN inbox_token TEXT;
CREATE UNIQUE INDEX idx_users_inbox_token ON users (inbox_token) WHERE inbox_token IS NOT NULL;

CREATE TABLE bank_emails (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id     TEXT NOT NULL,
    from_address   TEXT NOT NULL,
    subject        TEXT NOT NULL DEFAULT '',
    body           TEXT NOT NULL,
    received_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    occurred_at    TIMESTAMPTZ,
    status         TEXT NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','processing','imported','ignored','failed')),
    failure_reason TEXT NOT NULL DEFAULT '',
    processed_at   TIMESTAMPTZ,
    raw_body       TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX idx_bank_emails_user_message ON bank_emails (user_id, message_id);

ALTER TABLE transactions ADD COLUMN source TEXT NOT NULL DEFAULT 'manual'
      CHECK (source IN ('manual','email'));
ALTER TABLE transactions ADD COLUMN bank_email_id BIGINT
      REFERENCES bank_emails(id) ON DELETE SET NULL;

CREATE TABLE category_hints (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note_key    TEXT NOT NULL,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_category_hints_user_note ON category_hints (user_id, note_key);

CREATE TABLE bank_accounts (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_number TEXT NOT NULL,
    first_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_bank_accounts_user_number ON bank_accounts (user_id, account_number);
