# Thiết kế: tự động ghi giao dịch từ email biến động số dư

Ngày: 2026-08-28
Trạng thái: đã chốt qua brainstorming, chờ lập kế hoạch thực thi

## 1. Mục tiêu

Khi ngân hàng gửi email biến động số dư, app tự đọc email đó và tạo transaction
tương ứng, phân loại category dựa trên nội dung chuyển khoản. Người dùng không
phải nhập tay nữa; khi máy đoán sai category thì họ sửa, và lần sau máy đúng.

Phạm vi bản đầu: **MB Bank và TPBank**.

## 2. Những quyết định đã chốt

| Câu hỏi | Chốt | Vì sao |
| --- | --- | --- |
| Email vào app bằng đường nào | Người dùng đặt filter Gmail forward tới địa chỉ riêng do app cấp; Cloudflare Email Routing nhận, một Email Worker POST về app | App không giữ credential hộp thư của ai; là push nên chạy được trên Render free, không cần worker nền |
| Xử lý đồng bộ hay lưu trước | **Lưu email thô trước, xử lý sau** | Parser MB/TPBank chắc chắn sẽ sai vài lần. Email thô còn nằm đó thì sửa parser xong replay được; xử lý thẳng thì email đã bay và phải nhập tay |
| Transaction vào sổ thế nào | Ghi thẳng, tính vào số dư ngay, mang cờ `source='email'` | Đúng tinh thần tự động. Parse sai thì thiệt hại tối đa là một dòng sai, xoá được |
| Phân loại | **Tra bảng nhớ trước, gọi AI khi chưa gặp** | Nhất quán (cùng nội dung luôn ra cùng category) và học được từ người dùng; AI đơn thuần thì không có cả hai |
| Model | `claude-opus-5`, `effort: "low"` | Người dùng chọn trả phí sau khi cân với free tier của Google (free tier dùng nội dung để cải thiện sản phẩm — không hợp dữ liệu tài chính) |
| Giờ giao dịch | `transactions.occurred_on` giữ nguyên `DATE` | Toàn bộ app (lọc tháng, export, biểu đồ) xây trên ngày; thêm giờ là thay đổi lan khắp app. Giờ chính xác lưu trên `bank_emails.occurred_at` |
| Trùng với giao dịch nhập tay | Không tự đoán và bỏ qua; hiện dấu "có thể trùng" trên danh sách | Đoán sai nghĩa là mất một giao dịch mà người dùng không biết |
| Cùng một email tới hai lần | Chặn cứng bằng unique index trên `(user_id, message_id)` | Bắt buộc, không phải gợi ý |

## 3. Ranh giới package

Theo đúng khuôn `internal/csvimport` + `import_handlers.go`: phần đọc và phần
quyết định không chạm database, chỉ handler mới ghi.

| Nơi | Việc | Chạm DB? |
| --- | --- | --- |
| `internal/bankmail` | `(from, subject, body)` -> `Notice`. Mỗi ngân hàng một parser, chọn theo địa chỉ gửi. Kèm `NoteKey` chuẩn hoá nội dung CK | Không |
| `internal/classify` | Client gọi Claude: nội dung CK + danh sách category -> một category | Không |
| `internal/handlers/inbox_*.go` | Webhook, bảng nhớ, tạo transaction, trạng thái email | Có |

`inbox_` là prefix mới trong `internal/handlers`; thêm một dòng vào bảng prefix
trong CLAUDE.md.

Hai package đầu thuần hàm, nên mọi luật về đọc email và mọi luật về phân loại
test được không cần Postgres và không cần mạng.

## 4. Luồng dữ liệu

```
Gmail (filter: from MB/TPBank)
  --forward--> <token>@in.<domain>
      -- inbound parsing --POST--> /inbox/{token}
```

1. Webhook xác thực, lưu email thô vào `bank_emails` (status `pending`), trả 200.
   Không làm gì thêm trong request.
2. Kick goroutine với `context.Background()` — **không phải ctx của request**,
   vì request kết thúc ngay và ctx của nó bị huỷ theo (xem ghi chú đã có trong
   `auth_password_reset.go`).
3. Goroutine xử lý **mọi email `pending` của user đó**, không riêng email vừa
   tới. Nhờ vậy nếu Render restart bỏ lại một email dở thì email tiếp theo dọn
   hộ — không cần cron, không cần job khởi động.
4. Giành việc: `UPDATE bank_emails SET status='processing' WHERE id=$1 AND
   status='pending' RETURNING *`. Không trả row nghĩa là goroutine khác đã cầm.
5. `bankmail.Parse` -> `Notice`, hoặc lỗi (bảng ở mục 6).
6. `bankmail.NoteKey(Notice.Description)` -> tra `category_hints` của user.
   - Trúng: dùng luôn, không gọi AI.
   - Trượt: gọi `classify.Classify`, ghi kết quả vào `category_hints`.
   - AI lỗi hoặc chưa cấu hình: rơi về `other` / `other_income`. **Vẫn tạo
     transaction** — phân loại hỏng không được phép làm mất một giao dịch.
7. Insert transaction (`source='email'`, `bank_email_id` trỏ ngược lại), email
   chuyển `imported`, ghi `processed_at`.
8. Về sau, khi người dùng sửa category của một dòng `source='email'`, handler
   sửa hiện có ghi đè `category_hints` cho `note_key` đó. **Đây là chỗ nó học.**

## 5. Lược đồ — migration `000014_add_email_tracking`

Gộp tất cả vào một migration, theo tiền lệ `000013` (một tính năng = một
migration, dù chạm nhiều bảng).

### `bank_emails`

```sql
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
CREATE UNIQUE INDEX idx_bank_emails_user_message ON bank_emails (user_id, message_id);
```

- `message_id` lấy từ header `Message-ID`. Header này không phải lúc nào cũng
  có; thiếu thì handler băm `(from, subject, body)` làm khoá — cùng cách
  `Import.Fingerprint` trong `csvimport`.
- `body` chỉ lưu phần `text/plain`, cắt ở 64KB. Không lưu HTML, không lưu đính
  kèm. ~100 email/tháng trên Neon free (0.5GB) đủ dùng nhiều năm nên **không có
  cơ chế dọn dẹp** — thêm job prune bây giờ là giải quyết vấn đề chưa tồn tại.
- `processing` tồn tại chỉ để giành việc (bước 4 ở mục 4). Unique index trên
  `message_id` chặn *email* trùng, không chặn *xử lý* trùng.

### `category_hints`

```sql
CREATE TABLE category_hints (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note_key    TEXT NOT NULL,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_category_hints_user_note ON category_hints (user_id, note_key);
```

- `ON DELETE CASCADE` trên `category_id` **cố ý khác `transactions`**, vốn không
  có mệnh đề `ON DELETE` và được gán lại về `Other` khi xoá category. Một gợi ý
  trỏ vào category đã xoá là vô nghĩa; một giao dịch thì không được biến mất.
- **Cố tình không có cột `source ('ai'|'user')`.** Luật "AI chỉ ghi khi chưa có
  gợi ý, người dùng sửa thì luôn ghi đè" tự nó đủ; cột đó không đổi hành vi nào.

### `transactions`

```sql
ALTER TABLE transactions ADD COLUMN source TEXT NOT NULL DEFAULT 'manual'
      CHECK (source IN ('manual','email'));
ALTER TABLE transactions ADD COLUMN bank_email_id BIGINT
      REFERENCES bank_emails(id) ON DELETE SET NULL;
```

CHECK chỉ hai giá trị. Dòng từ CSV import vẫn là `'manual'` — thêm `'import'`
rồi không bao giờ ghi là code chết; phân biệt CSV là một thay đổi riêng.

### `users`

```sql
ALTER TABLE users ADD COLUMN inbox_token TEXT;
CREATE UNIQUE INDEX idx_users_inbox_token ON users (inbox_token) WHERE inbox_token IS NOT NULL;
```

`NULL` = chưa bật. Bật ở Settings thì sinh token — 32 byte ngẫu nhiên từ
`crypto/rand`, mã base64url không đệm — tắt thì về `NULL`. Đây cũng
là đường thu hồi: địa chỉ lộ thì tắt rồi bật lại, địa chỉ cũ chết ngay.

### Khôi phục `other_income`

```sql
INSERT INTO categories (user_id, name, type, color, slug)
SELECT NULL, 'Other income', 'income', '#A1A1AA', 'other_income'
WHERE NOT EXISTS (SELECT 1 FROM categories WHERE slug = 'other_income');
```

Migration này **đảo lại một quyết định của `000006`** và phải nói rõ vì sao
trong comment. `000006` xoá `Thu nhập khác` với lý do bộ 9 category mới "ghép
Lương với Thưởng thay cho một category thu nhập chung" — đúng vào lúc đó, vì
không ai cần một chỗ chứa thu nhập chung. Tính năng này thì cần: khi có tiền
vào mà AI không chắc, không có `other_income` nghĩa là không có chỗ nào để đặt
nó, và rơi về `Salary` sẽ ghi một khoản bạn bè trả nợ thành lương, làm hỏng
mọi đường so sánh tháng trên dashboard.

Insert theo kiểu "chỉ khi chưa có": tài khoản cũ còn giữ row `other_income`
(vì có transaction tham chiếu nên `000006` không xoá được) vẫn dùng row của họ.
Màu `#A1A1AA` là xám dành riêng cho "Other", hợp lệ với `categories_color_check`
của `000006`, và đúng nghĩa — cả hai đều là chỗ chứa cuối.

### Xoá tài khoản

`deleteAccount` liệt kê tường minh từng bảng thay vì phó mặc cascade. Thêm
`category_hints` và `bank_emails` vào chuỗi đó, `bank_emails` **sau**
`transactions` vì `transactions.bank_email_id` trỏ sang nó. Mở rộng test xoá
tài khoản hiện có để khẳng định không sót.

## 6. `internal/bankmail`

```go
type Notice struct {
    Bank        string    // "mb" | "tpbank"
    Amount      int64     // luôn dương, VND
    Direction   string    // "income" | "expense"
    OccurredAt  time.Time // giờ Việt Nam
    Description string    // nội dung CK thô, như ngân hàng viết
}

func Parse(from, subject, body string) (Notice, error)
```

Email ngân hàng có kèm số dư sau giao dịch, nhưng `Notice` **cố tình không giữ
nó**: không có chỗ nào trong app đọc tới, và một trường được parse rồi bỏ đó là
code chết. Khi nào cần đối soát số dư thì thêm sau, cùng với thứ dùng nó.

Lỗi trả về quyết định trạng thái email — đây là toàn bộ lý do có năm trạng thái:

| Lỗi | Nghĩa | `status` |
| --- | --- | --- |
| `ErrUnknownSender` | `from` không khớp MB hay TPBank | `ignored` |
| `ErrNotANotice` | đúng ngân hàng, nhưng là OTP/quảng cáo | `ignored` |
| lỗi khác | đúng loại email, parser đọc không ra | `failed` |

`ignored` là chuyện bình thường, không cần ai làm gì. `failed` là lỗi của mình
và là danh sách để replay sau khi sửa parser. Trộn hai cái làm một thì danh sách
"cần sửa" ngập email quảng cáo và không ai nhìn nó nữa.

Sentinel error `Err`-prefixed, so sánh bằng `errors.Is`.

**Không dùng lại `parseAmount` của `csvimport`.** Hàm đó cố ý dễ dãi (đoán `.`
hay `,` là dấu thập phân theo vị trí) vì nó đối mặt với file bất kỳ do người
dùng đưa vào. Email ngân hàng thì mỗi bank đúng một định dạng cố định; dễ dãi ở
đây nghĩa là khi ngân hàng đổi template ta được một **số tiền sai âm thầm** thay
vì một dòng `failed` nhìn thấy được và replay được. `bankmail` viết hàm đọc số
riêng, chặt.

### `NoteKey` — chỗ quyết định thành bại

```go
// NoteKey chuẩn hoá nội dung chuyển khoản thành khoá tra bảng nhớ.
func NoteKey(description string) string
```

Được export vì **hai nơi phải dùng đúng một luật**: lúc xử lý email (ghi gợi ý)
và lúc người dùng sửa category trên trang transactions (ghi đè gợi ý). Hai luật
lệch nhau một chút sẽ để lại những gợi ý không bao giờ tra trúng — y hệt lý do
`csvimport.MatchKey` được export.

Luật: thường hoá chữ, bỏ dấu tiếng Việt, **xoá mọi token chứa ít nhất một chữ
số** (không phải chỉ token toàn số — `FT24123456789` và `MBVCB.1234567890` đều
phải biến mất), gộp khoảng trắng.

Gạch đầu dòng thứ ba là quan trọng nhất. Nội dung CK gần như luôn kèm mã tham
chiếu duy nhất (`NGUYEN VAN A chuyen tien FT24123456789`, `Thanh toan GRAB
8829471`). Không xoá mã đó thì mỗi giao dịch là một khoá mới, bảng nhớ không bao
giờ trúng, và **mọi email đều gọi AI** — toàn bộ giá trị của thiết kế "nhớ
trước, AI sau" nằm ở hàm này.

## 7. `internal/classify`

Hình dạng sao chép `internal/mailer`, kể cả đường test.

```go
type Category struct {
    ID   int64
    Name string // đã qua i18n
}

func New(cfg Config) *Classifier
func (c *Classifier) Configured() bool
func (c *Classifier) Classify(ctx context.Context, note string, cats []Category) (int64, error)
```

- SDK chính thức `github.com/anthropics/anthropic-sdk-go`, không gọi HTTP thô.
  Test trỏ vào `httptest` bằng `option.WithBaseURL` — giữ đúng cái
  `mailer.NewWithEndpoint` đang cho mình.
- `claude-opus-5`, `output_config.effort: "low"`, `max_tokens: 256`. **Không tắt
  thinking**: trên Opus 5 tắt thinking có mấy kiểu hỏng khó chịu (model viết
  tool call vào text, rò thẻ `<thinking>`); hạ `effort` mới là cần gạt đúng.
- **Structured output** (`output_config.format`) để model trả đúng một id
  category, khỏi bóc chữ từ văn xuôi.
- Danh sách category đưa cho model là category thật của user — cả mặc định lẫn
  cái họ tự tạo — đã lọc theo `income`/`expense`. Model **không được tạo
  category mới**, chỉ chọn trong danh sách.
- **Không bật prompt caching**: prompt ở đây ~200 token, dưới ngưỡng tối thiểu
  để cache ăn; bật vào chỉ là code thừa.
- Lỗi bắt theo chuỗi bằng `errors.As` vào `*anthropic.Error`, switch trên
  `StatusCode`. Mọi lỗi đều rơi về category dự phòng, transaction vẫn được tạo.

## 8. Webhook

`POST /inbox/{token}` — **route công khai**, cùng nhóm `/healthz` và `/static/*`,
nằm ngoài `auth.RequireAuth` vì người gọi là Email Worker chứ không phải trình
duyệt có cookie.

Hai điều dễ quên:

- Route này phải **được miễn CSRF**. `csrf.Middleware` chặn mọi POST không mang
  token; Worker không có cookie nào để double-submit. Bỏ sót thì mọi email bị
  từ chối 403.
- Handler chỉ bóc `(from, subject, text, message-id)`, lưu, trả 200. **Phần bóc
  payload nằm riêng một file adapter** — đổi nhà cung cấp thì sửa đúng một file.
  Hình dạng payload là do Worker của mình định ra, nên adapter này mỏng; nó tồn
  tại để lần đổi sau không phải mổ handler.

### Ba lớp xác thực

1. Token trong địa chỉ là chuỗi ngẫu nhiên dài (map ra `users.inbox_token`).
2. Verify chữ ký HMAC do Worker ký bằng `INBOUND_WEBHOOK_SECRET` — chặn người
   gọi thẳng endpoint. Worker là code của mình nên chữ ký này là thứ mình tự
   đặt ra; xem ghi chú Brevo ở dưới để biết vì sao điều đó lại quan trọng.
3. **`From` của email gốc phải khớp địa chỉ đã biết của MB/TPBank.** Không khớp
   thì lưu lại nhưng không bao giờ tạo transaction (`ignored`).

Không có lớp 3 thì ai biết địa chỉ forward cũng gửi được email giả và giao dịch
giả vào thẳng sổ.

### Nhà cung cấp inbound — đã chốt: Cloudflare (2026-08-28)

**Cloudflare Email Routing + một Email Worker.** Email Routing miễn phí và
không giới hạn dung lượng thư; Worker chạy trên gói Workers Free. Catch-all cho
`in.ttth-caothang.site` nghĩa là mọi `<token>@in.ttth-caothang.site` vào thẳng
Worker mà không phải đăng ký từng địa chỉ — khớp đúng mô hình token mỗi user ở
mục 5. Giới hạn 25 MiB mỗi thư không đụng tới email ngân hàng.

Cái giá: một mẩu JS chỉ làm mỗi việc ký HMAC rồi POST sang app, và **domain
phải dùng nameserver của Cloudflare** — Email Routing đòi full setup, subdomain
chỉ bật được khi zone đã nằm ở Cloudflare. `ttth-caothang.site` đang ở Vietnix
nên phải đổi NS và dựng lại các bản ghi hiện có trước khi làm gì khác.

Từ 30/06/2025 Cloudflare bỏ im lặng thư forward từ domain nguồn không có SPF
hoặc DKIM. Nguồn ở đây là Gmail nên không thành vấn đề, nhưng khi một email
biến mất không dấu vết thì đây là chỗ nhìn đầu tiên.

**Brevo đã bị loại, dù tài khoản mail đi vẫn là Brevo.** Hai lý do, lý do sau
nặng hơn:

1. **Brevo không ký webhook.** Không HMAC, không header chữ ký, không shared
   secret; khuyến nghị chính thức của họ là allowlist dải IP. Lớp 2 ở trên sẽ
   không tồn tại, và mô hình một-URL-cho-cả-domain của họ còn buộc phải tách
   user từ trường `To` trong payload thay vì từ `/inbox/{token}`.
2. **Không xác nhận được trước khi đặt cược.** `GET /v3/webhooks` trả cùng một
   lỗi `document_not_found` cho mọi `type`, nên nó không nói được inbound có
   khả dụng hay không. Cách duy nhất để biết là verify domain và trỏ MX xong
   rồi thử tạo webhook — tức là làm hết phần DNS rồi mới biết có ăn hay không.
   Cloudflare thì tài liệu đã trả lời trước khi mình chạm vào thứ gì.

## 9. Cấu hình

Bốn biến, tất cả không bắt buộc — đúng cách `internal/config` đối xử với mọi thứ
trừ `DATABASE_URL`.

| Biến | Thiếu thì sao |
| --- | --- |
| `INBOUND_DOMAIN` | Thẻ Email tracking không hiện, tính năng tự tắt |
| `INBOUND_WEBHOOK_SECRET` | Webhook từ chối mọi request — không có secret thì không xác thực được ai gọi. Cùng một giá trị phải đặt vào secret của Worker bên Cloudflare; lệch nhau thì mọi email bị từ chối |
| `ANTHROPIC_API_KEY` | **Tính năng vẫn chạy đủ.** Không phân loại được thì rơi về `Other`/`Other income`, giao dịch vẫn vào sổ. Giống `BREVO_API_KEY` thiếu thì quên mật khẩu vẫn chạy tới bước gửi |
| `ANTHROPIC_MODEL` | Mặc định `claude-opus-5`. Có biến này để đổi sang Haiku sau mà không cần deploy code mới |

Cập nhật kèm: `.env.example` và `render.yaml` (`sync: false` cho hai cái mang bí
mật), và README giữ đồng bộ với CLAUDE.md.

## 10. UI — ba chỗ, không hơn

**1. Thẻ "Email tracking" trên `/settings`.** Trạng thái bật/tắt, địa chỉ
forward (kèm nút copy), đoạn hướng dẫn ngắn cách đặt filter Gmail, nút sinh lại
token (đường thu hồi nếu địa chỉ lộ). Ngay dưới: **tối đa 10 email `failed` gần
nhất** (tiêu đề + thời gian + lý do) kèm nút **"Thử lại"** đặt chúng về
`pending`. Nút đó là toàn bộ lý do chọn phương án lưu email thô.

**2. Nhãn `auto`** trên dòng giao dịch có `source='email'`, cạnh phần mô tả.
Cộng dấu **"có thể trùng"** khi rơi trúng cùng ngày + số tiền + loại với một
dòng đã có. Không cần cột mới và **không cần query mới**: trang transactions đã
tải sẵn các dòng của trang đó, nên dấu này tính khi dựng danh sách, bằng cách so
các dòng đang có trong tay với nhau. Hệ quả phải chấp nhận: hai dòng trùng nhau
nhưng nằm ở hai trang khác nhau thì không được đánh dấu — đổi lại không tốn
thêm truy vấn nào cho một gợi ý mà người dùng vẫn phải tự quyết.

**3. Không có UI mới cho việc học.** Người dùng sửa category của một dòng `auto`
thì handler sửa hiện tại ghi thêm vào `category_hints`. Việc học diễn ra bằng
thao tác họ vốn đã làm.

Template mới tự động chịu các test bất biến trong `view_layout_test.go` — không
hex, không `rgba(`, không class Tailwind lạc ra ngoài `class=""`.

## 11. Test

| Ở đâu | Cần Postgres? | Khẳng định gì |
| --- | --- | --- |
| `bankmail` | Không | Bảng test dựng từ mẫu email thật (đã che tên/số TK): số tiền, chiều tiền, ngày giờ, nội dung CK. Cả ca xấu: OTP -> `ErrNotANotice`, người gửi lạ -> `ErrUnknownSender`. Phần dày nhất |
| `bankmail.NoteKey` | Không | Bảng riêng: hai nội dung chỉ khác mã tham chiếu phải ra cùng một khoá |
| `classify` | Không | `httptest` giả Anthropic: request gửi đi đúng model/effort/format; 429 và 500 trả lỗi chứ không panic |
| `internal/handlers/inbox_*_test.go` | Có | Email -> transaction; email trùng không tạo hai dòng; `failed` không tạo gì; gợi ý ghi rồi tra trúng (lần hai không gọi AI); sửa category ghi đè gợi ý; xoá tài khoản dọn sạch cả hai bảng mới |

Không test nào cần mạng thật hay khoá API thật.

## 12. Thứ tự làm

Chỉ `bankmail` bị chặn bởi mẫu email. Mọi thứ khác làm được ngay.

1. Xác minh Brevo Inbound Parsing có trong gói miễn phí không (chặn mục 8)
2. Migration `000014` + `sqlc generate`
3. Webhook + lưu email + adapter payload
4. Thẻ settings (bật/tắt, địa chỉ, danh sách `failed`, nút thử lại)
5. `internal/classify`
6. `internal/bankmail` (khi có mẫu email)
7. Nối đầu cuối: xử lý pending, bảng nhớ, tạo transaction, nhãn `auto`, học từ
   thao tác sửa category

**Lưu ý vận hành:** test handler không tự chạy migration, nên khi có `000014`
phải áp nó vào database test cục bộ trước, không thì test DB mới sẽ đỏ vì thiếu
bảng.

## 13. Đã cân nhắc và loại

- **IMAP polling / Gmail API OAuth** cho ingestion: cái đầu bắt app giữ
  credential đọc được toàn bộ hộp thư; cái sau bắt qua verification restricted
  scope của Google, quá nặng cho một app cá nhân.
- **Brevo Inbound Parsing** cho ingestion: không ký webhook, và không xác nhận
  được có dùng được hay không cho tới khi đã làm xong phần DNS — lý do đầy đủ ở
  mục 8.
- **Gemini free tier** cho phân loại: free tier dùng nội dung để cải thiện sản
  phẩm, không hợp với nội dung chuyển khoản.
- **Chỉ keyword, không AI**: nội dung CK tiếng Việt không dấu viết tắt thì
  keyword thua xa.
- **Hộp chờ duyệt**: an toàn nhưng vẫn phải thao tác từng cái, mất lý do tự động hoá.
- **Cột `source` trên `category_hints`**: không đổi hành vi nào.
- **Job dọn `bank_emails`**: giải quyết vấn đề chưa tồn tại.
- **Thêm giờ vào `transactions.occurred_on`**: thay đổi lan khắp app, không đáng.
