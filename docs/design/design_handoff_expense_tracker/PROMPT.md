# Prompt để dán vào Claude Code

> Copy toàn bộ phần trong khung dưới đây. Đặt hai file `SPEC.md` và `Expense Tracker UI.dc.html`
> vào thư mục `docs/design/` trong repo trước khi chạy (hoặc sửa đường dẫn trong prompt).

---

Tôi đang xây một web app expense tracker cá nhân (dùng cho tôi và bạn bè/gia đình, mỗi người
một tài khoản riêng, KHÔNG có tính năng chia tiền nhóm). Tôi đã có mockup giao diện và cần bạn
triển khai vào codebase này.

## Tài liệu thiết kế

- `docs/design/SPEC.md` — đặc tả đầy đủ: design tokens, từng màn hình, empty state, responsive.
  Đây là nguồn sự thật. Đọc hết file này trước khi viết code.
- `docs/design/Expense Tracker UI.dc.html` — mockup HTML để tham khảo trực quan (mở trong browser
  để xem). **Đây là file design reference, KHÔNG phải production code.** Đừng copy markup từ đó
  sang; nó dùng inline style và một runtime riêng. Hãy đọc nó để hiểu bố cục, khoảng cách, thứ bậc
  thị giác — rồi viết lại bằng Go html/template + Tailwind theo đúng convention của repo này.

Mockup là **high-fidelity**: màu, font, spacing, kích cỡ chữ trong SPEC.md là con số cuối cùng,
hãy làm sát. Nếu repo đã có component/partial tương đương (button, input, card), dùng lại của repo
thay vì tạo mới, miễn là kết quả thị giác khớp với spec.

## Stack

- Go, `html/template` server-rendered
- htmx cho tương tác không reload trang
- Tailwind CSS
- Chart.js cho biểu đồ (Tổng quan)
- Chỉ VND, format `50.000₫` — dấu chấm phân cách nghìn, không thập phân, ký hiệu ₫ ở cuối
- Toàn bộ chữ giao diện: tiếng Việt

## Việc cần làm

Trước khi code, hãy đọc codebase và cho tôi biết:
1. Cấu trúc hiện tại: router, layout template, chỗ đặt partial, cách build CSS, đã có auth chưa,
   schema DB hiện có.
2. Kế hoạch triển khai theo từng bước, mỗi bước là một đơn vị có thể review được.

Sau khi tôi xác nhận kế hoạch thì mới bắt đầu code. Thứ tự tôi muốn:

1. **Nền tảng** — Tailwind config (màu, font, radius theo SPEC.md), layout template gốc,
   nav dùng chung (top bar desktop / bottom bar mobile), helper format tiền VND cho template
   (`{{ vnd .Amount }}`), helper format ngày.
2. **Auth** — trang đăng nhập/đăng ký, session cookie, middleware bảo vệ các trang sau đăng nhập.
3. **Danh mục** — model + seed 9 danh mục mặc định (danh sách trong SPEC.md), trang Danh mục,
   thêm/sửa/xóa qua htmx. Danh mục mặc định KHÔNG xóa được — chặn cả ở handler, không chỉ ẩn nút.
4. **Giao dịch** — model, trang chính, form thêm nhanh (htmx prepend row mới), lọc theo tháng,
   sửa/xóa inline.
5. **Tổng quan** — 2 thẻ số liệu, biểu đồ tròn theo danh mục, biểu đồ cột 4 tháng gần nhất.
   Truyền data cho Chart.js qua `<script type="application/json">` chứ đừng nhúng JS vào template.
6. **Responsive pass** — kiểm lại từng trang ở 390px và 1280px.

## Yêu cầu kỹ thuật

- Mọi mutation (thêm/sửa/xóa) trả về HTML fragment cho htmx swap, không trả JSON, không full reload.
- Mọi query đều scope theo `user_id`. Người dùng A không được đọc/sửa dữ liệu của B —
  kiểm tra quyền sở hữu trong handler, không dựa vào việc UI không hiện link.
- Số tiền lưu dạng integer (đơn vị đồng), không dùng float.
- CSRF token cho mọi form POST. Mật khẩu hash bằng bcrypt. Đăng xuất là POST, không phải GET link.
- Validate ở server: số tiền > 0, danh mục thuộc về user, loại giao dịch khớp loại danh mục,
  ngày không ở tương lai xa. Lỗi trả về fragment form kèm thông báo tiếng Việt, giữ nguyên
  dữ liệu người dùng đã nhập.
- Mỗi màn hình đều phải có empty state đúng như SPEC.md — đây là app mới nên empty state là
  thứ người dùng thấy đầu tiên, không phải trường hợp ngoại lệ.
- Viết test cho: format tiền VND, chặn xóa danh mục mặc định, scope theo user, tính tổng thu/chi
  theo tháng.

## Ràng buộc

- Đừng thêm tính năng tôi không yêu cầu (budget, mục tiêu tiết kiệm, import CSV, dark mode,
  đa tiền tệ, chia tiền nhóm).
- Đừng thêm build step hay dependency mới ngoài stack trên nếu không hỏi tôi trước.
- Không icon library — nav và các nút dùng chữ, đúng như mockup. Nếu cần một icon nhỏ
  (mũi tên dropdown, dấu ＋) thì dùng ký tự hoặc SVG inline.
- Giữ tối giản: bảng màu trung tính, một màu accent duy nhất cho nút chính và trạng thái active.
  Màu danh mục chỉ xuất hiện dưới dạng chấm 8–10px và trong biểu đồ, không dùng làm nền.

Bắt đầu bằng việc đọc codebase và SPEC.md, rồi trình bày kế hoạch.
