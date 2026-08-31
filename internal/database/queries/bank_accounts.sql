-- RememberBankAccount ghi nhận một số tài khoản là của người dùng. Không lỗi
-- khi đã có: mỗi email lặp lại đều gọi hàm này.
-- name: RememberBankAccount :exec
INSERT INTO bank_accounts (user_id, account_number)
VALUES ($1, $2)
ON CONFLICT (user_id, account_number) DO NOTHING;

-- BankAccountBelongsToUser trả về true khi số tài khoản này đã từng xuất
-- hiện ở ô "Tài khoản trích nợ" của chính người dùng đó.
-- name: BankAccountBelongsToUser :one
SELECT EXISTS (
    SELECT 1 FROM bank_accounts WHERE user_id = $1 AND account_number = $2
) AS owns;
