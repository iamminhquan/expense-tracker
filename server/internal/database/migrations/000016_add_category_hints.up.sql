-- category_hints la bo nho note_key -> category ma buoc xu ly email tra
-- truoc khi roi ve Other/Other income, va nguoi dung ghi de moi khi ho sua
-- category cua mot dong source='email'. Day la toan bo co che "hoc" cua
-- tinh nang.
CREATE TABLE category_hints (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note_key    TEXT NOT NULL,
    -- ON DELETE CASCADE co y khac transactions.category_id, von khong co
    -- menh de ON DELETE nao ca: mot gợi y tro vao category da bi xoa la vo
    -- nghia va nen bien mat theo, con mot transaction thi khong bao gio duoc
    -- phep bien mat chi vi category cua no bi xoa.
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Moi user chi co mot gợi y cho mot note_key: lan sua sau ghi de lan truoc
-- qua ON CONFLICT (user_id, note_key) DO UPDATE, khong bao gio cong don.
CREATE UNIQUE INDEX idx_category_hints_user_note ON category_hints (user_id, note_key);

-- Co tinh khong co cot source ('ai'|'user'). Luat "AI chi ghi khi chua co
-- gợi y, nguoi dung sua thi luon ghi de" tu no da du; mot cot nhu vay khong
-- doi hanh vi nao ca.
