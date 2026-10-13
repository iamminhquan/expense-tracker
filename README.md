# $pend

Ứng dụng web ghi chép thu chi cá nhân, tính bằng VND. Nhập khoản chi trong vài giây, cuối tháng
xem tiền đã đi đâu.

[![CI](https://github.com/iamminhquan/expense-tracker/actions/workflows/ci.yml/badge.svg)](https://github.com/iamminhquan/expense-tracker/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Dùng thử:** <https://expenseboard.vercel.app>. API chạy trên gói miễn phí của Render, nên lần
mở đầu tiên có thể phải chờ vài giây để server khởi động.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/dashboard-dark.png">
  <img alt="Trang Overview: tổng chi tháng, biểu đồ chi tiêu theo danh mục và biểu đồ thu chi 4 tháng gần nhất" src="docs/screenshots/dashboard-light.png">
</picture>

## Giới thiệu

$pend là sổ thu chi dành cho một người hoặc từng thành viên trong gia đình. Mỗi tài khoản có dữ
liệu riêng: không chia hóa đơn, không ngân sách chung, không ai xem được sổ của người khác. Số tiền
là VND nguyên (không có phần lẻ), giao diện bằng tiếng Anh.

Mọi tài khoản dùng chung một bộ danh mục mặc định (Food & Drink, Transport, Bills, Salary, …), và
mỗi người có thể thêm danh mục riêng với màu tự chọn. Ứng dụng làm tốt cả trên điện thoại lẫn máy
tính: điện thoại có thanh tab nổi, nút "+" để thêm nhanh và cử chỉ nhấn giữ; máy tính có bảng
giao dịch đầy đủ với form nhập ngay trên trang.

## Tính năng

- **Sổ thu chi**: thêm, sửa, xóa khoản thu và khoản chi; lọc theo tháng, danh mục, loại hoặc tìm theo
  ghi chú; phân trang; tự đánh dấu giao dịch có vẻ bị nhập trùng.
- **Tổng quan tháng**: đã chi, đã thu, chênh lệch và so sánh với tháng trước; biểu đồ tròn theo
  danh mục; biểu đồ cột thu chi 4 tháng gần nhất; số dư hiện tại cộng dồn qua các tháng.
- **Danh mục**: danh mục mặc định dùng chung, cộng danh mục riêng với 8 màu để chọn.
- **Nhập và xuất CSV**: tải file lên, ghép cột, xem trước rồi mới nhập. Nếu có dòng lỗi thì không dòng
  nào được nhập, để dữ liệu không bị nhập dở dang. Xuất toàn bộ giao dịch ra CSV bất cứ lúc nào.
- **Tài khoản**: đăng ký, đăng nhập, xác thực email, quên mật khẩu, đổi email và mật khẩu, xem và đăng
  xuất các phiên đăng nhập trên thiết bị khác, xóa tài khoản cùng toàn bộ dữ liệu.
- **Giao diện**: chế độ sáng, tối hoặc theo hệ thống; bố cục mobile-first với bottom sheet và cử chỉ
  nhấn giữ để sửa hoặc xóa.
- **Bảo mật**: access token JWT ngắn hạn chỉ giữ trong bộ nhớ, refresh token nằm trong cookie httpOnly
  và đổi mới sau mỗi lần dùng; khóa đăng nhập 15 phút sau 5 lần sai mật khẩu; giới hạn số request
  theo IP.

<p align="center">
  <img alt="Danh sách giao dịch trên máy tính, có form thêm giao dịch và bộ lọc" src="docs/screenshots/transactions.png">
</p>

<table align="center">
  <tr>
    <td><img alt="Trang Overview trên điện thoại, giao diện sáng" src="docs/screenshots/mobile-dashboard.png" width="280"></td>
    <td><img alt="Danh sách giao dịch trên điện thoại, giao diện tối" src="docs/screenshots/mobile-transactions-dark.png" width="280"></td>
  </tr>
</table>

## Công nghệ

| Phần | Công nghệ |
| --- | --- |
| Frontend (`client/`) | React 19, TypeScript, Vite, Tailwind CSS v4, TanStack Query, React Router, Chart.js |
| Backend (`server/`) | Go 1.26, Gin, pgx, sqlc, golang-migrate |
| Cơ sở dữ liệu | PostgreSQL |
| Hạ tầng | Vercel (frontend), Render (backend), Neon (Postgres), Brevo (email) |

## Kiến trúc

```mermaid
flowchart LR
    B[Trình duyệt]
    B -- "tải ứng dụng" --> V["client/<br/>React SPA trên Vercel"]
    B -- "/api/v1/* + JWT" --> R["server/<br/>Go API trên Render"]
    R --> N[("PostgreSQL<br/>trên Neon")]
    R -. "email xác thực, đặt lại mật khẩu" .-> M[Brevo]
```

Repo là một monorepo gồm hai package độc lập. `server/` là một JSON API viết bằng Go, không render
HTML; `client/` là một single-page app React, chỉ nói chuyện với server qua `/api/v1/*`. Câu SQL được
viết tay rồi sinh code Go bằng `sqlc`. Server tự chạy migration mỗi lần khởi động, nên không có bước
migrate riêng, kể cả khi deploy.

## Chạy trên máy

Cần có Go 1.26+, PostgreSQL 16+, Node.js 22+ và [pnpm](https://pnpm.io).

**1. Backend.** Tạo một database trống (ví dụ `createdb expense_tracker`), rồi:

```bash
cd server
cp .env.example .env
```

Trong `.env`, chỉ cần điền hai biến bắt buộc:

```bash
DATABASE_URL=postgres://USER:PASSWORD@localhost:5432/expense_tracker?sslmode=disable
JWT_SECRET=   # tạo bằng: openssl rand -base64 32
```

```bash
go run ./cmd/server   # API chạy ở http://localhost:8080
```

Lần chạy đầu, server tự tạo bảng và các danh mục mặc định. Những biến còn lại đều có giá trị mặc định
và được giải thích ngay trong `.env.example`. Nếu để trống `BREVO_API_KEY`, email xác thực và đặt lại
mật khẩu sẽ không được gửi đi (server chỉ ghi log rằng đã bỏ qua), còn mọi thứ khác vẫn chạy bình
thường.

**2. Frontend.** Mở một terminal khác:

```bash
cd client
pnpm install
pnpm dev   # mở http://localhost:5173
```

Vite chuyển tiếp các request `/api/*` sang `http://localhost:8080`, nên không cần cấu hình CORS khi
chạy trên máy.

**Kiểm thử.** Các test có đụng tới database sẽ bị bỏ qua nếu không đặt `TEST_DATABASE_URL`. Nên trỏ
biến này tới một database riêng cho test, vì bộ test sẽ ghi dữ liệu vào đó:

```bash
cd server
export TEST_DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/expense_tracker_test?sslmode=disable"
go test -run '^TestRunMigrations$' ./cmd/server   # tạo schema trước
go test ./...

cd ../client
pnpm lint && pnpm test && pnpm build
```

## Cấu trúc thư mục

```text
.
├── server/              Go API: handler, xác thực, migration và câu SQL
├── client/              React SPA: các trang, component, hook, API client
├── docs/screenshots/    Ảnh chụp màn hình dùng trong README
├── .github/workflows/   CI: build, lint và test cả hai package trước khi merge
└── .claude/             Tài liệu kiến trúc và quy ước code của từng package
```

Muốn tìm hiểu sâu hơn (luồng xác thực, schema, quy ước API, những quyết định thiết kế), xem
[`.claude/context/server.md`](.claude/context/server.md) và
[`.claude/context/client.md`](.claude/context/client.md).

## Triển khai

Mỗi lần push lên `main`, Render tự build và deploy `server/` (cấu hình trong `server/render.yaml`),
còn Vercel deploy `client/` (cấu hình trong `client/vercel.json`). Hai bên deploy độc lập với nhau.
Database chạy trên Neon; migration mới được áp dụng ngay khi server bản mới khởi động.

## Giấy phép

Phát hành theo [giấy phép MIT](LICENSE).
