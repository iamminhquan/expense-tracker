-- Email thô được lưu trước, xử lý sau. Parser MB/TPBank chắc chắn sẽ sai vài
-- lần; email còn nằm đây thì sửa parser xong replay được, còn xử lý thẳng thì
-- thư đã bay và người dùng phải nhập tay.
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
    processed_at   TIMESTAMPTZ
);

-- Chặn cứng email trùng. Không phải gợi ý: cùng một thư tới hai lần là chuyện
-- bình thường của forward, và nó không được thành hai giao dịch.
CREATE UNIQUE INDEX idx_bank_emails_user_message ON bank_emails (user_id, message_id);

-- Địa chỉ nhận riêng của mỗi tài khoản. NULL = chưa bật. Tắt rồi bật lại sinh
-- token mới, và đó cũng là đường thu hồi khi địa chỉ bị lộ.
ALTER TABLE users ADD COLUMN inbox_token TEXT;
CREATE UNIQUE INDEX idx_users_inbox_token ON users (inbox_token) WHERE inbox_token IS NOT NULL;

-- CHECK chỉ hai giá trị. Dòng từ CSV import vẫn là 'manual': thêm 'import' rồi
-- không bao giờ ghi là code chết, và phân biệt CSV là một thay đổi riêng.
ALTER TABLE transactions ADD COLUMN source TEXT NOT NULL DEFAULT 'manual'
      CHECK (source IN ('manual','email'));
ALTER TABLE transactions ADD COLUMN bank_email_id BIGINT
      REFERENCES bank_emails(id) ON DELETE SET NULL;

-- Đảo lại một quyết định của 000006, cố ý. 000006 xoá 'Thu nhập khác' vì bộ 9
-- category mới ghép Lương với Thưởng thay cho một category thu nhập chung --
-- đúng vào lúc đó, vì không ai cần một chỗ chứa thu nhập chung. Tính năng này
-- thì cần: khi có tiền vào mà máy không chắc, không có other_income nghĩa là
-- không có chỗ nào đặt nó, và rơi về Salary sẽ ghi một khoản bạn bè trả nợ
-- thành lương, làm hỏng mọi đường so sánh tháng trên dashboard.
--
-- Insert theo kiểu "chỉ khi chưa có": tài khoản cũ còn giữ row other_income
-- (vì có transaction tham chiếu nên 000006 không xoá được) vẫn dùng row của họ.
INSERT INTO categories (user_id, name, type, color, slug)
SELECT NULL, 'Other income', 'income', '#A1A1AA', 'other_income'
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE slug = 'other_income');
