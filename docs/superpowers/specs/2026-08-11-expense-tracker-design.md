# Expense Tracker — Design

## Mục tiêu

Xây dựng một web app expense tracker đơn giản cho cá nhân, bạn bè và gia đình dùng. Mỗi người có tài khoản riêng và theo dõi thu chi độc lập (không chia sẻ/chia tiền giữa các tài khoản trong bản đầu).

## 1. Kiến trúc tổng thể & Tech stack

Monolith Go, server-rendered HTML (không tách frontend/backend riêng).

- **Router:** `chi` — nhẹ, idiomatic, đủ middleware cần thiết (logging, recover, session).
- **Data layer:** `sqlc` — viết SQL tay, sinh Go code type-safe. Migration bằng `golang-migrate`.
- **Database:** PostgreSQL.
- **Template/UI:** `html/template` chuẩn của Go + `htmx` (tương tác không cần reload trang, ví dụ: thêm/sửa/xóa giao dịch, lọc theo tháng) + Tailwind CSS (qua CDN lúc đầu cho đơn giản) cho styling.
- **Biểu đồ báo cáo:** Chart.js (CDN) — render biểu đồ tròn/cột phía client từ dữ liệu JSON server trả về.
- **Auth:** Đăng ký/đăng nhập bằng email + mật khẩu (bcrypt hash). Session lưu trong PostgreSQL (bảng `sessions`), cookie `httpOnly` + `secure` chứa session ID — đơn giản, dễ revoke, không cần lo JWT expiry/refresh.
- **Tiền tệ:** Chỉ VND. Số tiền lưu dưới dạng số nguyên (đơn vị đồng), không cần xử lý thập phân/quy đổi.
- **Triển khai:** Dockerfile + docker-compose (Go app + Postgres) để chạy local ngay, sẵn sàng deploy lên VPS khi cần (chưa quyết định môi trường deploy cụ thể).

## 2. Data model

```
users
  id, email (unique), password_hash, name, created_at

sessions
  id (token), user_id, expires_at

categories
  id, user_id (NULL = default category dùng chung cho mọi user), name, type (expense|income), icon/color, created_at

transactions
  id, user_id, category_id, amount (integer, đơn vị đồng, luôn dương), type (expense|income), description, occurred_on (date), created_at, updated_at
```

- Mỗi user chỉ thấy giao dịch/danh mục của chính mình, cộng với các danh mục mặc định (`user_id IS NULL`).
- Seed sẵn danh mục mặc định khi migrate: Ăn uống, Di chuyển, Giải trí, Hóa đơn, Sức khỏe, Mua sắm (expense); Lương, Thu nhập khác (income).

## 3. Tính năng & luồng chính

- **Auth:** Đăng ký → đăng nhập → session cookie → middleware bắt buộc login cho mọi trang trừ `/login`, `/register`.
- **Giao dịch:** Trang danh sách giao dịch (lọc theo tháng), form thêm/sửa (htmx modal hoặc inline), xóa có xác nhận.
- **Danh mục:** Trang quản lý danh mục cá nhân (thêm/sửa/xóa danh mục do user tự tạo; không xóa/sửa được danh mục mặc định).
- **Báo cáo:** Trang dashboard hiển thị tổng thu/chi tháng hiện tại, biểu đồ tròn theo danh mục, biểu đồ cột so sánh vài tháng gần nhất.

## 4. Error handling

- Lỗi validate (số tiền âm/không hợp lệ, thiếu trường bắt buộc...) trả về form kèm thông báo lỗi qua htmx (không reload trang).
- Lỗi hệ thống (DB down...) → trang lỗi chung, log chi tiết phía server.
- Chống truy cập chéo dữ liệu: mọi query lọc theo `user_id` lấy từ session, không tin tưởng ID trên URL/form.

## 5. Testing

- Unit test cho các hàm nghiệp vụ (tính tổng, validate) bằng package `testing` chuẩn của Go.
- Integration test cho các endpoint chính (auth, CRUD giao dịch, báo cáo) dùng test database (Postgres qua Docker hoặc `testcontainers-go`).

## Ngoài phạm vi (out of scope) cho bản đầu

- Chia tiền/nhóm chi tiêu chung giữa nhiều người (kiểu Splitwise).
- Ngân sách/hạn mức chi tiêu và cảnh báo vượt mức.
- Đa tiền tệ.
- OAuth/Google login.
- Mobile app riêng.
