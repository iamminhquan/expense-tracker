# Lát 2 — Detect giao dịch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Email ngân hàng đã lưu ở lát 1 tự biến thành transaction trong sổ, mang cờ `source='email'`, rơi về category `other`/`other_income`. Hết nhập tay.

**Architecture:** `internal/bankmail` là package thuần hàm biến `(from, subject, body)` thành `Notice`, không chạm database và không chạm mạng. `internal/handlers/inbox_process.go` là nơi duy nhất ghi: nó giành việc trên `bank_emails` bằng một `UPDATE ... RETURNING`, gọi `bankmail.Parse`, tạo transaction, rồi đóng trạng thái email. Webhook của lát 1 kick goroutine sau khi lưu.

**Tech Stack:** Go, pgx/v5 + sqlc, chi, html/template + htmx.

**Spec:** `docs/superpowers/specs/2026-08-28-bank-email-tracking-design.md` — mục 4, 6, 10, 11.

## Global Constraints

- Tiền là **số nguyên VND**. `(VND) 20,000.00` là hai mươi nghìn đồng, không phải hai mươi nghìn xu. Phần thập phân làm tròn.
- Mọi mốc thời gian theo `Asia/Ho_Chi_Minh` — dùng `vietnamLocation` sẵn có trong `internal/handlers/req_month.go`. `transactions.occurred_on` là `DATE`; giờ chính xác ghi vào `bank_emails.occurred_at`.
- `gofmt -l .` không in gì; `go vet ./...`, `go build ./...` sạch.
- Test dùng `testing` chuẩn, không testify. Mặc định `package foo_test`. Test chạm DB đọc `TEST_DATABASE_URL` và `t.Skip` khi trống; lấy URL từ `DATABASE_URL` trong `.env`.
- Thông điệp lỗi test: `FuncName(input) = got, want want`, `%q` cho chuỗi.
- Error chữ thường, không dấu chấm. Sentinel `Err`-prefixed, so bằng `errors.Is`.
- Comment giải thích **vì sao**, không phải cái gì.
- File trong `internal/handlers` mang tiền tố nhóm; phần xử lý dùng `inbox_`.
- Không `<style>`/`<script>` nội tuyến trong template; không màu hex; class Tailwind phải nằm trong `class="..."`.
- Commit subject 72–100 ký tự, có tiền tố `<type>: `.
- **Không hand-edit `internal/sqlcgen/`.** `sqlc` ở `/home/minhquan/go/bin/sqlc`, không nằm trong PATH.

## Phạm vi bị thu hẹp vì thiếu mẫu

Chỉ có mẫu MB, khuôn **"Chuyển tiền nội bộ MB"**, và **chỉ chiều chi**. Không có mẫu TPBank, không có mẫu tiền vào.

Do đó:
- `Parse` nhận diện MB qua địa chỉ gửi và qua khuôn thư. Khuôn lạ → `ErrNotANotice` → email thành `failed`, nằm lại để replay khi có mẫu.
- **Không viết parser TPBank.** Một parser đoán mò còn tệ hơn không có: nó tạo giao dịch sai vào sổ tiền thật.
- Chiều tiền suy từ cấu trúc: có `Tài khoản trích nợ` nghĩa là tiền ra. Không có → `ErrNotANotice`, không đoán.

---

## File Structure

**Tạo mới:**
- `internal/bankmail/bankmail.go` — `Notice`, `Parse`, sentinel errors.
- `internal/bankmail/mb.go` — parser khuôn MB.
- `internal/bankmail/notekey.go` — `NoteKey`.
- `internal/bankmail/*_test.go` — không cần Postgres.
- `internal/handlers/inbox_process.go` — goroutine xử lý `pending`.
- `internal/handlers/inbox_process_test.go`.

**Sửa:**
- `internal/database/queries/bank_emails.sql` — giành việc, đóng trạng thái, đếm trùng.
- `internal/database/queries/transactions.sql` — nếu cần cột `source`/`bank_email_id` khi tạo.
- `internal/handlers/inbox_webhook.go` — kick goroutine sau khi lưu.
- `internal/web/templates/transaction_row.html` — nhãn `auto`, dấu trùng.
- `internal/handlers/txn_*.go` — dựng cờ cho mỗi dòng.
- `CLAUDE.md`.

---

### Task 1: `internal/bankmail` — Notice, Parse, và khuôn MB

**Files:**
- Create: `internal/bankmail/bankmail.go`, `internal/bankmail/mb.go`
- Test: `internal/bankmail/bankmail_test.go`, `internal/bankmail/mb_test.go`

**Interfaces:**
- Produces:
  - `type Notice struct { Bank string; Amount int64; Direction string; OccurredAt time.Time; Description string }`
  - `func Parse(from, subject, body string) (Notice, error)`
  - `var ErrUnknownSender = errors.New("sender is not a bank this app reads")`
  - `var ErrNotANotice = errors.New("not a transaction notice")`

**Fixture bắt buộc dùng — đây là email MB thật, đã che số tài khoản.** Chú ý nhãn bị ngắt dòng giữa chừng (`Số\n tiền giao dịch`): đó là bảng HTML xuống dòng, và nó là lý do parser phải gộp khoảng trắng trước khi khớp nhãn.

```
Cảm ơn Quý khách đã sử dụng dịch vụ MB eBanking. 

MB xin thông báo giao dịch của Quý khách đã được thực hiện như sau:

 
 
 Ngày,
 giờ giao dịch 

 
 
 31-08-2026 01:05:33 

 
 

 
 
 Loại
 giao dịch 

 
 
 Chuyển tiền nội bộ MB 

 
 

 
 
 Số
 tham chiếu 

 
 
 26083101055223730 

 
 

 
 
 Tài
 khoản trích nợ 

 
 
 NGUYEN VAN A - 0001111111111 (VND) 

 
 

 
 
 Người
 thụ hưởng 

 
 
 Nguyen Van B - 0399999999 

 
 

 
 
 Số
 tiền giao dịch 

 
 
 (VND) 20,000.00 

 
 

 
 
 Nội
 dung chuyển tiền 

 
 
 NGUYEN VAN A chuyen tien 

 
 

 
 
 Cách
 thức lệnh 

 
 
 Thanh toán ngay 

 
 

 
 
 Tình
 trạng 

 
 
 Giao dịch thành công 
```

- [ ] **Step 1: Viết test thất bại từ fixture thật**

Đặt fixture trên vào một hằng chuỗi trong `mb_test.go`. Test khẳng định:

```go
n, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich thanh cong", mbTransferNotice)
// err == nil
// n.Bank == "mb"
// n.Amount == 20000            // 20,000.00 VND -> 20000 dong, khong phai 2000000
// n.Direction == "expense"
// n.OccurredAt == 2026-08-31 01:05:33 +07
// n.Description == "NGUYEN VAN A chuyen tien"
```

Thêm các ca xấu, mỗi ca một `t.Run`:
- người gửi lạ (`someone@example.com`) → `errors.Is(err, bankmail.ErrUnknownSender)`
- đúng MB nhưng thân thư là quảng cáo/OTP (không có nhãn nào) → `ErrNotANotice`
- đúng khuôn nhưng `Tình trạng` không phải `Giao dịch thành công` → `ErrNotANotice` (giao dịch hỏng không được vào sổ)
- thiếu `Tài khoản trích nợ` → `ErrNotANotice` (không đoán chiều tiền)
- số tiền `(VND) 1,234,567.89` → `Amount == 1234568` (làm tròn)

- [ ] **Step 2: Chạy để chắc nó fail**

Run: `go test ./internal/bankmail/...` → FAIL, package chưa tồn tại.

- [ ] **Step 3: Implement**

Yêu cầu thiết kế, không phải code sẵn:
- Nhận MB qua hậu tố `@mbbank.com.vn` trên địa chỉ gửi, dùng lại đúng luật so hậu tố kèm `@` như `isKnownBankSender` (xem `internal/handlers/inbox_webhook.go`). Người gửi lạ → `ErrUnknownSender` **trước** khi đọc thân thư.
- Gộp mọi chuỗi khoảng trắng (kể cả `\n`) thành một dấu cách trước khi khớp nhãn. Không làm bước này thì `Số\n tiền giao dịch` không bao giờ khớp.
- Đọc số tiền **riêng, chặt**, không dùng lại `csvimport.parseAmount`. MB có đúng một định dạng; dễ dãi ở đây nghĩa là ngân hàng đổi khuôn thì ta được một **số tiền sai âm thầm** thay vì một dòng `failed` nhìn thấy được. Luật: bỏ dấu phẩy ngăn nghìn, phần sau dấu chấm là thập phân, làm tròn về đồng.
- Ngày giờ: `31-08-2026 01:05:33` theo `Asia/Ho_Chi_Minh`.
- `Description` lấy giá trị sau nhãn `Nội dung chuyển tiền`, **dừng trước nhãn kế tiếp**. Đừng nuốt cả phần còn lại của thư.
- `Notice` **không** giữ số dư sau giao dịch dù thư có: một trường được parse rồi bỏ đó là code chết.

- [ ] **Step 4: Chạy test** → PASS toàn bộ.

- [ ] **Step 5: KHÔNG commit.** Báo lại cho controller.

---

### Task 2: `NoteKey`

**Files:** Create `internal/bankmail/notekey.go`, `internal/bankmail/notekey_test.go`

**Interfaces:** Produces `func NoteKey(description string) string`

`NoteKey` chuẩn hoá nội dung chuyển khoản thành khoá tra bảng nhớ ở lát 3. Nó được export vì **hai nơi phải dùng đúng một luật**: lúc xử lý email và lúc người dùng sửa category. Hai luật lệch nhau sẽ để lại những gợi ý không bao giờ tra trúng.

Luật:
1. thường hoá chữ
2. bỏ dấu tiếng Việt
3. **xoá mọi token chứa ít nhất một chữ số** — không phải chỉ token toàn số
4. gộp khoảng trắng

- [ ] **Step 1: Viết test bảng thất bại**

Ca bắt buộc:
- `"NGUYEN VAN A chuyen tien"` → `"nguyen van a chuyen tien"`
- Hai nội dung chỉ khác mã tham chiếu phải ra **cùng một khoá**: `"NGUYEN VAN A chuyen tien FT24123456789"` và `"NGUYEN VAN A chuyen tien MBVCB.9876543210"` → cùng `"nguyen van a chuyen tien"`
- `"Thanh toan GRAB 8829471"` → `"thanh toan grab"`
- `"Chuyển tiền học phí"` → `"chuyen tien hoc phi"` (bỏ dấu)
- chuỗi rỗng → chuỗi rỗng

Gạch đầu dòng 3 là quan trọng nhất: nội dung CK gần như luôn kèm mã tham chiếu duy nhất. Không xoá thì mỗi giao dịch là một khoá mới, bảng nhớ không bao giờ trúng, và **mọi email đều gọi AI** — toàn bộ giá trị của thiết kế nằm ở hàm này.

- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: KHÔNG commit.**

---

### Task 3: Truy vấn cho vòng xử lý

**Files:** Modify `internal/database/queries/bank_emails.sql`, regenerate `internal/sqlcgen`

**Interfaces:** Produces
- `ClaimPendingBankEmail(ctx, id) (BankEmail, error)` — `UPDATE ... SET status='processing' WHERE id=$1 AND status='pending' RETURNING *`
- `ListPendingBankEmailIDs(ctx, userID) ([]int64, error)`
- `MarkBankEmailImported(ctx, MarkBankEmailImportedParams{ID, OccurredAt}) error` — set `status='imported'`, `processed_at=now()`
- `MarkBankEmailFailed(ctx, MarkBankEmailFailedParams{ID, FailureReason}) error`
- `MarkBankEmailIgnored(ctx, MarkBankEmailIgnoredParams{ID, FailureReason}) error`
- `CountDuplicateTransactions(ctx, ...) (int64, error)` — đếm transaction cùng user, cùng ngày, cùng số tiền, cùng loại

Giành việc bằng `UPDATE ... WHERE status='pending' RETURNING` là điều bắt buộc: không trả dòng nào nghĩa là goroutine khác đã cầm. Đừng đọc rồi mới ghi.

- [ ] Viết query, chạy `PATH="$PATH:/home/minhquan/go/bin" sqlc generate`, `go build ./...`.
- [ ] **KHÔNG commit.**

---

### Task 4: Vòng xử lý — email thành transaction

**Files:** Create `internal/handlers/inbox_process.go`, `internal/handlers/inbox_process_test.go`; Modify `internal/handlers/inbox_webhook.go`

**Interfaces:** Produces `func processPendingEmails(deps Deps, userID int64)`

Luồng, theo đúng mục 4 của spec:
1. Webhook lưu email xong, trả 200, rồi kick goroutine. **Dùng `context.Background()`, không dùng ctx của request** — request kết thúc ngay và ctx của nó bị huỷ theo. Xem ghi chú sẵn có trong `auth_password_reset.go`.
2. Goroutine xử lý **mọi** email `pending` của user đó, không riêng email vừa tới. Nhờ vậy nếu Render restart bỏ lại một email dở thì email tiếp theo dọn hộ — không cần cron.
3. Mỗi email: giành việc → `bankmail.Parse` → tạo transaction → đóng trạng thái.

Ánh xạ lỗi → trạng thái:
| Lỗi | `status` | Vì sao |
| --- | --- | --- |
| `ErrUnknownSender` | `ignored` | chuyện bình thường, không ai cần làm gì |
| `ErrNotANotice` | `ignored` | đúng ngân hàng nhưng là OTP/quảng cáo |
| lỗi khác | `failed` | lỗi của mình, và là danh sách để replay |

Trộn hai loại làm một thì danh sách "cần sửa" ngập email quảng cáo và không ai nhìn nó nữa.

Tạo transaction:
- `source='email'`, `bank_email_id` trỏ ngược lại email
- category rơi về slug `other` (chi) hoặc `other_income` (thu) — tra theo **slug**, không bao giờ theo tên hiển thị
- `occurred_on` = phần ngày của `Notice.OccurredAt`; giờ đầy đủ ghi vào `bank_emails.occurred_at`
- note = `Notice.Description`, cắt 200 ký tự cho khớp luật của `handleCreateTransaction`

- [ ] **Step 1: Viết test thất bại** (cần Postgres). Ca bắt buộc:
  - email MB hợp lệ → đúng **một** transaction, `source='email'`, số tiền đúng, category `other`
  - email từ người gửi lạ → không transaction nào, email `ignored`
  - thân thư không phải thông báo → không transaction nào, email `ignored`
  - chạy vòng xử lý **hai lần** trên cùng email → vẫn đúng một transaction (giành việc có tác dụng)
  - email `failed` không tạo gì và giữ nguyên `failure_reason`
- [ ] **Step 2–4:** fail → implement → pass.
- [ ] **Step 5: KHÔNG commit.**

---

### Task 5: Nhãn `auto` và dấu "có thể trùng"

**Files:** Modify `internal/web/templates/transaction_row.html`, và nơi dựng dòng trong `internal/handlers/txn_*.go`; Test trong `internal/handlers`

Theo mục 10 phần 2 của spec:
- Dòng có `source='email'` mang nhãn **`auto`** cạnh phần mô tả.
- Thêm dấu **"có thể trùng"** khi dòng đó rơi trúng cùng ngày + cùng số tiền + cùng loại với một dòng khác **đang hiển thị trên cùng trang**.
- **Không thêm truy vấn mới.** Trang transactions đã tải sẵn các dòng của trang đó, nên dấu này tính khi dựng danh sách, bằng cách so các dòng đang có trong tay với nhau.
- Hệ quả phải chấp nhận: hai dòng trùng nhau nằm ở hai trang khác nhau thì không được đánh dấu. Đổi lại không tốn thêm truy vấn nào cho một gợi ý mà người dùng vẫn phải tự quyết. Ghi điều này vào comment.

- [ ] Viết test trước (dựng danh sách có hai dòng trùng → cả hai mang cờ; hai dòng khác ngày → không cờ nào).
- [ ] Implement, chạy cả `view_layout_test.go`.
- [ ] **KHÔNG commit.**

---

### Task 6: Tài liệu

**Files:** Modify `CLAUDE.md`

- [ ] Thêm `internal/bankmail` vào đoạn Email ingestion: nó biến email thành `Notice`, thuần hàm, và **`NoteKey` được export vì hai nơi phải dùng chung một luật**.
- [ ] Ghi rõ giới hạn hiện tại: chỉ MB, chỉ khuôn chuyển tiền, chỉ chiều chi; TPBank và chiều thu chưa có mẫu nên chưa viết, và một email lạ khuôn sẽ thành `failed` để replay chứ không bị đoán bừa.
- [ ] CLAUDE.md để prose ở **dòng dài**, không tự xuống hàng.
- [ ] **KHÔNG commit.**

---

## Self-Review

**Spec coverage:** mục 6 (`bankmail`, `NoteKey`) → Task 1–2; mục 4 (luồng, giành việc, ánh xạ lỗi) → Task 3–4; mục 10 phần 2 (nhãn `auto`, dấu trùng) → Task 5; mục 11 (test) → rải khắp.

**Cố ý để lại cho lát sau:** `category_hints` và việc học (lát 3, migration `000016` vì `000015` đã dùng cho `raw_body`), `internal/classify` (lát 4), parser TPBank và khuôn tiền vào (chờ mẫu).

**Rủi ro lớn nhất:** parser dựng trên **một** mẫu duy nhất. Mọi khuôn MB khác sẽ thành `failed` — đó là hành vi đúng và replay được, nhưng đừng nới `Parse` cho dễ dãi để "đỡ failed": một số tiền sai âm thầm đắt hơn nhiều một dòng failed nhìn thấy được.
