-- Các số tài khoản ngân hàng mà app biết chắc là của người dùng.
--
-- Nguồn duy nhất để ghi vào đây là ô "Tài khoản trích nợ" của một email đã
-- gửi tới hộp thư riêng của tài khoản đó: MB chỉ gửi thông báo trừ tiền cho
-- chính chủ tài khoản, nên một số xuất hiện ở ô ấy là bằng chứng, không phải
-- phỏng đoán. Trùng tên người thụ hưởng thì KHÔNG được ghi vào đây -- hai
-- người có thể trùng tên, và đoán sai ở bảng này khiến một khoản chi thật
-- biến mất khỏi sổ mà chủ tài khoản không có cách nào biết.
CREATE TABLE bank_accounts (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_number TEXT NOT NULL,
    first_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Mỗi số chỉ ghi một lần cho mỗi người. Email lặp lại là chuyện thường của
-- forward, và nó không được đẻ ra hàng trăm dòng giống nhau.
CREATE UNIQUE INDEX idx_bank_accounts_user_number ON bank_accounts (user_id, account_number);
