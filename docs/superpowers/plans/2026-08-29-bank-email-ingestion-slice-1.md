# Lát 1 — Ingestion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Người dùng bật Email tracking trong Settings, forward email ngân hàng tới địa chỉ app cấp, và **nhìn thấy email đó nằm trong Settings** — chưa tạo transaction nào.

**Architecture:** Cloudflare Email Worker (`emailworker/`) nhận thư, ký HMAC-SHA256 lên thân request rồi POST JSON sang `POST /inbox/{token}` — route công khai, miễn CSRF. Handler xác thực ba lớp (token → chữ ký → `From`), lưu email thô vào `bank_emails` với `status='pending'`, trả 200 và không làm gì thêm. Mọi luật thuần hàm (hình dạng payload, ký/verify, sinh token, fingerprint) nằm ở `internal/inbound`, không chạm request và không chạm database, theo đúng tiền lệ `internal/csvimport`.

**Tech Stack:** Go 1.x + chi + pgx/v5 + sqlc, golang-migrate, html/template + htmx, Cloudflare Workers (JS, Web Crypto API), wrangler 4.127.1.

**Spec:** `docs/superpowers/specs/2026-08-28-bank-email-tracking-design.md` (mục 4, 5, 8, 9, 10, 11)

## Global Constraints

- Tiền là số nguyên VND, không thập phân. UI tiếng Anh; chỉ tên category đi qua `internal/i18n`.
- `gofmt -l .` phải không in gì; `go vet ./...` sạch; `go build ./...` xanh — trước **mọi** commit.
- Test dùng `testing` chuẩn, không testify. Mặc định `package handlers_test`; cần identifier chưa export thì `package handlers` và đặt tên file `*_internal_test.go`.
- Test chạm DB đọc `TEST_DATABASE_URL` và `t.Skip` khi trống.
- **Test handler không tự chạy migration.** Phải `migrate` tay `000014` vào database test cục bộ trước, không thì test đỏ vì thiếu bảng.
- File trong `internal/handlers` mang tiền tố theo nhóm; `inbox_` là tiền tố mới.
- Không `<style>`/`<script>` nội tuyến trong template. Không hex, không `rgba(`, không class Tailwind lạc ngoài `class="..."` — `view_layout_test.go` fail trên các thứ đó.
- Lỗi chữ thường không dấu chấm; bọc bằng `%w` khi caller có thể cần inspect.
- Commit: subject **72–100 ký tự** kể cả tiền tố `<type>: `. Không footer "Generated with Claude Code".
- Sửa `emailworker/` thì **phải** `npx wrangler deploy` từ trong thư mục đó — `git push` không đụng tới nó.

---

## File Structure

**Tạo mới:**
- `internal/database/migrations/000014_add_email_tracking.up.sql` / `.down.sql` — bảng `bank_emails`, `users.inbox_token`, `transactions.source`/`bank_email_id`, khôi phục `other_income`.
- `internal/database/queries/bank_emails.sql` — truy vấn của bảng mới.
- `internal/inbound/inbound.go` — `Payload`, `ParsePayload`, `Sign`, `Verify`, `NewToken`, `Fingerprint`, `TruncateBody`. Thuần hàm.
- `internal/inbound/inbound_test.go` — test không cần Postgres.
- `internal/handlers/inbox_webhook.go` — `inboxWebhookHandler`, ba lớp xác thực, lưu email.
- `internal/handlers/inbox_webhook_test.go` — test chạm DB.
- `internal/handlers/settings_inbox.go` — bật/tắt/sinh lại token, danh sách `failed`, nút thử lại. Nằm ở `settings_` vì nó render vào trang settings; phần webhook mới mang tiền tố `inbox_`.
- `internal/handlers/settings_inbox_test.go`
- `internal/web/templates/settings_inbox.html` — thẻ "Email tracking".

**Sửa:**
- `internal/config/config.go` — thêm `InboundDomain`, `InboundWebhookSecret`.
- `internal/config/config_test.go`
- `cmd/server/main.go` — truyền hai giá trị mới vào `Deps`.
- `internal/handlers/app_deps.go` — hai trường mới.
- `internal/handlers/app_router.go` — route `/inbox/{token}` miễn CSRF + 3 route settings.
- `internal/handlers/settings_handlers.go` — `settingsData` thêm dữ liệu thẻ mới; `deleteAccount` xoá `bank_emails`.
- `internal/web/web.go` — thêm `settings_inbox.html` vào page set `settings`.
- `internal/web/templates/settings.html` — chèn thẻ mới.
- `emailworker/src/index.js` — bản thật thay bản log.
- `emailworker/wrangler.toml` — bỏ đoạn "bản tạm".
- `.env.example`, `render.yaml`, `README.md`, `CLAUDE.md`.

---

### Task 1: Migration 000014 và truy vấn sinh ra từ nó

**Files:**
- Create: `internal/database/migrations/000014_add_email_tracking.up.sql`
- Create: `internal/database/migrations/000014_add_email_tracking.down.sql`
- Create: `internal/database/queries/bank_emails.sql`
- Modify: `internal/database/queries/users.sql` (thêm 2 truy vấn ở cuối)
- Regenerate: `internal/sqlcgen/` (không sửa tay)

**Interfaces:**
- Consumes: không có.
- Produces: `sqlcgen.BankEmail`; `Queries.CreateBankEmail(ctx, CreateBankEmailParams) (BankEmail, error)`; `Queries.ListRecentFailedBankEmails(ctx, ListRecentFailedBankEmailsParams) ([]BankEmail, error)`; `Queries.RequeueFailedBankEmails(ctx, int64) error`; `Queries.DeleteBankEmailsForUser(ctx, int64) error`; `Queries.SetInboxToken(ctx, SetInboxTokenParams) error`; `Queries.GetUserByInboxToken(ctx, pgtype.Text) (User, error)`.

- [ ] **Step 1: Viết migration up**

Tạo `internal/database/migrations/000014_add_email_tracking.up.sql`:

```sql
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
```

- [ ] **Step 2: Viết migration down**

Tạo `internal/database/migrations/000014_add_email_tracking.down.sql`:

```sql
ALTER TABLE transactions DROP COLUMN bank_email_id;
ALTER TABLE transactions DROP COLUMN source;
DROP INDEX IF EXISTS idx_users_inbox_token;
ALTER TABLE users DROP COLUMN inbox_token;
DROP TABLE bank_emails;
-- Category other_income cố ý ở lại: có thể đã có transaction trỏ vào nó, và
-- transactions.category_id không có ON DELETE clause nào.
```

- [ ] **Step 3: Viết truy vấn cho bảng mới**

Tạo `internal/database/queries/bank_emails.sql`:

```sql
-- CreateBankEmail lưu một thư vừa tới. ON CONFLICT DO NOTHING vì unique index
-- trên (user_id, message_id) là thứ chặn thư trùng, và một thư trùng không
-- phải lỗi để trả về cho Worker -- Worker không làm gì được với nó.
-- name: CreateBankEmail :one
INSERT INTO bank_emails (user_id, message_id, from_address, subject, body, status, failure_reason)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, message_id) DO NOTHING
RETURNING *;

-- name: ListRecentFailedBankEmails :many
SELECT * FROM bank_emails
WHERE user_id = $1 AND status = 'failed'
ORDER BY received_at DESC
LIMIT $2;

-- RequeueFailedBankEmails là nút "Thử lại": đặt mọi thư failed về pending để
-- lượt xử lý sau đọc lại chúng bằng parser đã sửa. Đây là toàn bộ lý do chọn
-- phương án lưu email thô.
-- name: RequeueFailedBankEmails :exec
UPDATE bank_emails SET status = 'pending', failure_reason = '', processed_at = NULL
WHERE user_id = $1 AND status = 'failed';

-- name: CountBankEmailsForUser :one
SELECT count(*) FROM bank_emails WHERE user_id = $1;

-- name: DeleteBankEmailsForUser :exec
DELETE FROM bank_emails WHERE user_id = $1;
```

- [ ] **Step 4: Thêm hai truy vấn user vào cuối `internal/database/queries/users.sql`**

```sql
-- SetInboxToken bật hoặc tắt địa chỉ nhận email của tài khoản. NULL = tắt, và
-- tắt rồi bật lại sinh token khác, nên đây cũng là đường thu hồi.
-- name: SetInboxToken :exec
UPDATE users SET inbox_token = $2 WHERE id = $1;

-- GetUserByInboxToken là lớp xác thực thứ nhất của webhook: token trong địa chỉ
-- phải map ra đúng một tài khoản.
-- name: GetUserByInboxToken :one
SELECT * FROM users WHERE inbox_token = $1;
```

- [ ] **Step 5: Áp migration vào database test rồi sinh lại sqlc**

```bash
cd /home/minhquan/projects/go/expense_tracker
migrate -path internal/database/migrations -database "$TEST_DATABASE_URL" up
sqlc generate
```

Nếu không có CLI `migrate`, chạy `go run ./cmd/server` một lần với `DATABASE_URL` trỏ vào database test — server tự áp migration lúc khởi động — rồi Ctrl+C.

- [ ] **Step 6: Kiểm tra build và migration test**

Run: `go build ./... && go test ./internal/database/... -v`
Expected: PASS. `internal/sqlcgen/` có `BankEmail` và 5 method mới.

- [ ] **Step 7: Commit**

```bash
git add internal/database internal/sqlcgen
git commit -m "feat: add the email tracking schema and bring back the other_income fallback category"
```

---

### Task 2: `internal/inbound` — payload, chữ ký, token

**Files:**
- Create: `internal/inbound/inbound.go`
- Test: `internal/inbound/inbound_test.go`

**Interfaces:**
- Consumes: không có.
- Produces:
  - `type Payload struct { From, To, Subject, MessageID, Text string }`
  - `func ParsePayload(b []byte) (Payload, error)`
  - `func Sign(secret string, body []byte) string`
  - `func Verify(secret string, body []byte, signature string) bool`
  - `func NewToken() (string, error)`
  - `func TokenFromAddress(addr string) string`
  - `func Fingerprint(p Payload) string`
  - `func TruncateBody(s string) string`
  - `const MaxBodyBytes = 64 * 1024`
  - `var ErrEmptyPayload = errors.New("payload has no sender or body")`

- [ ] **Step 1: Viết test thất bại**

Tạo `internal/inbound/inbound_test.go`:

```go
package inbound_test

import (
	"strings"
	"testing"

	"expensetracker/internal/inbound"
)

func TestSignAndVerifyRoundTrip(t *testing.T) {
	body := []byte(`{"from":"a@b.c"}`)
	sig := inbound.Sign("s3cret", body)
	if !inbound.Verify("s3cret", body, sig) {
		t.Errorf("Verify(secret, body, Sign(...)) = false, want true")
	}
}

func TestVerifyRejectsWrongSecretBodyOrSignature(t *testing.T) {
	body := []byte(`{"from":"a@b.c"}`)
	sig := inbound.Sign("s3cret", body)

	cases := []struct {
		name           string
		secret, sigArg string
		body           []byte
	}{
		{"wrong secret", "other", sig, body},
		{"tampered body", "s3cret", sig, []byte(`{"from":"evil@x.y"}`)},
		{"garbage signature", "s3cret", "not-hex", body},
		{"empty signature", "s3cret", "", body},
		{"empty secret", "", sig, body},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if inbound.Verify(tc.secret, tc.body, tc.sigArg) {
				t.Errorf("Verify(%q, %q, %q) = true, want false", tc.secret, tc.body, tc.sigArg)
			}
		})
	}
}

func TestSignIsStableAcrossCalls(t *testing.T) {
	body := []byte("abc")
	if a, b := inbound.Sign("k", body), inbound.Sign("k", body); a != b {
		t.Errorf("Sign twice = %q, %q, want equal", a, b)
	}
}

func TestParsePayloadReadsTheWorkerShape(t *testing.T) {
	raw := []byte(`{"from":"no-reply@mb.com.vn","to":"abc123@in.example.site",
		"subject":"Bien dong so du","messageId":"<m1@mail>","text":"body here"}`)
	got, err := inbound.ParsePayload(raw)
	if err != nil {
		t.Fatalf("ParsePayload() error = %v, want nil", err)
	}
	if got.From != "no-reply@mb.com.vn" {
		t.Errorf("From = %q, want %q", got.From, "no-reply@mb.com.vn")
	}
	if got.MessageID != "<m1@mail>" {
		t.Errorf("MessageID = %q, want %q", got.MessageID, "<m1@mail>")
	}
	if got.Text != "body here" {
		t.Errorf("Text = %q, want %q", got.Text, "body here")
	}
}

func TestParsePayloadRejectsEmptyEnvelope(t *testing.T) {
	if _, err := inbound.ParsePayload([]byte(`{"from":"","text":""}`)); err == nil {
		t.Error("ParsePayload(empty envelope) error = nil, want error")
	}
}

func TestTokenFromAddress(t *testing.T) {
	cases := []struct{ in, want string }{
		{"abc123@in.example.site", "abc123"},
		{"ABC123@IN.EXAMPLE.SITE", "abc123"},
		{"  abc123@in.example.site  ", "abc123"},
		{"Someone <abc123@in.example.site>", "abc123"},
		{"no-at-sign", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := inbound.TokenFromAddress(tc.in); got != tc.want {
			t.Errorf("TokenFromAddress(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNewTokenIsLongAndUnique(t *testing.T) {
	a, err := inbound.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v, want nil", err)
	}
	b, _ := inbound.NewToken()
	if a == b {
		t.Errorf("NewToken() twice = %q both times, want different", a)
	}
	if len(a) < 40 {
		t.Errorf("len(NewToken()) = %d, want >= 40", len(a))
	}
	if strings.ContainsAny(a, "+/=") {
		t.Errorf("NewToken() = %q, want base64url without padding", a)
	}
}

func TestFingerprintDependsOnEveryField(t *testing.T) {
	base := inbound.Payload{From: "a@b.c", Subject: "s", Text: "t"}
	same := inbound.Fingerprint(base)
	if same != inbound.Fingerprint(base) {
		t.Error("Fingerprint is not stable for the same payload")
	}
	for _, changed := range []inbound.Payload{
		{From: "z@b.c", Subject: "s", Text: "t"},
		{From: "a@b.c", Subject: "z", Text: "t"},
		{From: "a@b.c", Subject: "s", Text: "z"},
	} {
		if inbound.Fingerprint(changed) == same {
			t.Errorf("Fingerprint(%+v) = same as base, want different", changed)
		}
	}
}

func TestTruncateBodyCapsAtMaxBodyBytes(t *testing.T) {
	long := strings.Repeat("x", inbound.MaxBodyBytes+500)
	if got := len(inbound.TruncateBody(long)); got != inbound.MaxBodyBytes {
		t.Errorf("len(TruncateBody(long)) = %d, want %d", got, inbound.MaxBodyBytes)
	}
	if got := inbound.TruncateBody("short"); got != "short" {
		t.Errorf("TruncateBody(%q) = %q, want unchanged", "short", got)
	}
}
```

- [ ] **Step 2: Chạy test để chắc nó fail**

Run: `go test ./internal/inbound/...`
Expected: FAIL — package `inbound` chưa tồn tại.

- [ ] **Step 3: Viết implementation**

Tạo `internal/inbound/inbound.go`:

```go
// Package inbound holds the rules the email ingestion path shares with the
// Cloudflare Email Worker in emailworker/: the payload the Worker sends, the
// signature it stamps on it, and the token that names which account a
// forwarded message belongs to.
//
// Nothing here touches a request or the database, which is what lets every
// one of those rules be tested without Postgres -- the same split
// internal/csvimport draws between reading a file and writing rows.
package inbound

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// MaxBodyBytes caps the stored plain-text body. Bank notices are a few KB;
// the cap exists so one oversized forward cannot fill the free-tier database.
const MaxBodyBytes = 64 * 1024

// SignatureHeader carries the hex HMAC the Worker stamps on the request body.
const SignatureHeader = "X-Inbox-Signature"

// ErrEmptyPayload is returned when a payload carries neither a sender nor a
// body, which means the Worker sent something that cannot be a bank notice.
var ErrEmptyPayload = errors.New("payload has no sender or body")

// Payload is the JSON the Email Worker POSTs. The field names are the
// contract: change one here and emailworker/src/index.js changes with it, in
// the same commit, or email silently stops arriving.
type Payload struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	MessageID string `json:"messageId"`
	Text      string `json:"text"`
}

// ParsePayload decodes the Worker's JSON body.
func ParsePayload(b []byte) (Payload, error) {
	var p Payload
	if err := json.Unmarshal(b, &p); err != nil {
		return Payload{}, fmt.Errorf("decode payload: %w", err)
	}
	if strings.TrimSpace(p.From) == "" && strings.TrimSpace(p.Text) == "" {
		return Payload{}, ErrEmptyPayload
	}
	return p, nil
}

// Sign returns the hex HMAC-SHA256 of body under secret.
//
// The signature covers the body alone and carries no timestamp: replaying a
// captured request cannot create a second row, because bank_emails has a
// unique index on (user_id, message_id) and a replay carries the same
// Message-ID it did the first time.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether signature is Sign(secret, body). An empty secret
// never verifies: a deployment that forgot to configure one must reject every
// request rather than accept every request.
func Verify(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	want := Sign(secret, body)
	return subtle.ConstantTimeCompare([]byte(want), []byte(signature)) == 1
}

// NewToken returns the random local part of an account's inbox address: 32
// bytes from crypto/rand, base64url with no padding, so it survives being
// pasted into an email address unescaped.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// TokenFromAddress pulls the local part out of a recipient address, which is
// how a delivered message names the account it belongs to. It accepts a bare
// address or a "Name <addr>" form, and returns "" for anything that is not an
// address at all.
func TokenFromAddress(addr string) string {
	s := strings.TrimSpace(addr)
	if i := strings.LastIndex(s, "<"); i >= 0 {
		s = strings.TrimSuffix(s[i+1:], ">")
	}
	at := strings.Index(s, "@")
	if at <= 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(s[:at]))
}

// Fingerprint is the dedupe key for a message whose Message-ID header is
// missing -- not every forwarder sets one, and the unique index still needs a
// value. Same idea as csvimport's Import.Fingerprint: a digest of what was
// actually read.
func Fingerprint(p Payload) string {
	h := sha256.New()
	// Length-prefixed so that moving text between fields cannot collide.
	for _, part := range []string{p.From, p.Subject, p.Text} {
		fmt.Fprintf(h, "%d:%s", len(part), part)
	}
	return "fp-" + hex.EncodeToString(h.Sum(nil))
}

// TruncateBody caps a body at MaxBodyBytes.
func TruncateBody(s string) string {
	if len(s) <= MaxBodyBytes {
		return s
	}
	return s[:MaxBodyBytes]
}
```

- [ ] **Step 4: Chạy test**

Run: `go test ./internal/inbound/... -v`
Expected: PASS toàn bộ.

- [ ] **Step 5: Commit**

```bash
gofmt -l . && go vet ./...
git add internal/inbound
git commit -m "feat: add internal/inbound for the payload, signature and token the Worker shares with the app"
```

---

### Task 3: Cấu hình `INBOUND_DOMAIN` và `INBOUND_WEBHOOK_SECRET`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `cmd/server/main.go`
- Modify: `internal/handlers/app_deps.go`
- Modify: `.env.example`, `render.yaml`

**Interfaces:**
- Consumes: không có.
- Produces: `config.Config.InboundDomain string`, `config.Config.InboundWebhookSecret string`; `handlers.Deps.InboundDomain string`, `handlers.Deps.InboundWebhookSecret string`.

- [ ] **Step 1: Viết test thất bại**

Thêm vào `internal/config/config_test.go`:

```go
func TestLoadReadsInboundSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("INBOUND_DOMAIN", "in.example.site")
	t.Setenv("INBOUND_WEBHOOK_SECRET", "shh")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.InboundDomain != "in.example.site" {
		t.Errorf("InboundDomain = %q, want %q", cfg.InboundDomain, "in.example.site")
	}
	if cfg.InboundWebhookSecret != "shh" {
		t.Errorf("InboundWebhookSecret = %q, want %q", cfg.InboundWebhookSecret, "shh")
	}
}

func TestLoadLeavesInboundSettingsEmptyWhenUnset(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	t.Setenv("INBOUND_DOMAIN", "")
	t.Setenv("INBOUND_WEBHOOK_SECRET", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.InboundDomain != "" || cfg.InboundWebhookSecret != "" {
		t.Errorf("inbound settings = %q/%q, want both empty", cfg.InboundDomain, cfg.InboundWebhookSecret)
	}
}
```

- [ ] **Step 2: Chạy test để chắc nó fail**

Run: `go test ./internal/config/...`
Expected: FAIL — `cfg.InboundDomain` undefined.

- [ ] **Step 3: Thêm hai trường vào `Config` và `Load`**

Trong `internal/config/config.go`, thêm vào struct `Config` sau `MailFrom`:

```go
	// InboundDomain is the domain forwarded bank email is received at
	// (e.g. "in.example.site"). Empty means the feature is off: the
	// settings card hides itself, because an inbox address cannot be built
	// without it.
	InboundDomain string
	// InboundWebhookSecret is the HMAC secret the Cloudflare Email Worker
	// signs its POST with. Empty rejects every webhook request rather than
	// accepting every one -- there is nothing to authenticate a caller with.
	// The same value must be set as the Worker's secret; see
	// emailworker/wrangler.toml.
	InboundWebhookSecret string
```

Và trong `Load`, sau `MailFrom`:

```go
		InboundDomain:        getEnv("INBOUND_DOMAIN", ""),
		InboundWebhookSecret: getEnv("INBOUND_WEBHOOK_SECRET", ""),
```

- [ ] **Step 4: Chạy test**

Run: `go test ./internal/config/... -v`
Expected: PASS.

- [ ] **Step 5: Truyền xuống Deps**

Trong `internal/handlers/app_deps.go`, thêm vào cuối struct `Deps`:

```go
	// InboundDomain and InboundWebhookSecret configure the email ingestion
	// path; see internal/config.Config for what an empty value means.
	InboundDomain        string
	InboundWebhookSecret string
```

Trong `cmd/server/main.go`, thêm hai dòng vào literal `handlers.Deps{...}`:

```go
		InboundDomain:        cfg.InboundDomain,
		InboundWebhookSecret: cfg.InboundWebhookSecret,
```

- [ ] **Step 6: Ghi vào `.env.example`**

Thêm vào cuối:

```
# Domain that forwarded bank email is received at, e.g. in.example.site.
# Leave empty to keep the Email tracking card hidden and the feature off.
INBOUND_DOMAIN=
# Shared HMAC secret the Cloudflare Email Worker signs its POST with. Must
# match the Worker's own secret (`npx wrangler secret put
# INBOUND_WEBHOOK_SECRET` from emailworker/). Empty rejects every webhook.
INBOUND_WEBHOOK_SECRET=
```

- [ ] **Step 7: Ghi vào `render.yaml`**

Thêm vào danh sách `envVars`, theo đúng kiểu `BREVO_API_KEY` đã dùng:

```yaml
      # Set by hand in the Render dashboard alongside the Cloudflare Worker's
      # own secret -- the two must be the same string or every email is
      # rejected with 403.
      - key: INBOUND_WEBHOOK_SECRET
        sync: false
      - key: INBOUND_DOMAIN
        sync: false
```

- [ ] **Step 8: Build và commit**

Run: `go build ./... && go test ./internal/config/... && gofmt -l . && go vet ./...`
Expected: PASS, `gofmt -l .` không in gì.

```bash
git add internal/config cmd/server internal/handlers/app_deps.go .env.example render.yaml
git commit -m "feat: read the inbound domain and webhook secret from the environment"
```

---

### Task 4: Webhook `POST /inbox/{token}` — ba lớp xác thực và lưu email thô

**Files:**
- Create: `internal/handlers/inbox_webhook.go`
- Test: `internal/handlers/inbox_webhook_test.go`
- Modify: `internal/handlers/app_router.go`

**Interfaces:**
- Consumes: `inbound.Verify`, `inbound.ParsePayload`, `inbound.Fingerprint`, `inbound.TruncateBody`, `inbound.SignatureHeader`, `inbound.MaxBodyBytes`; `Queries.GetUserByInboxToken`, `Queries.CreateBankEmail`.
- Produces: `func inboxWebhookHandler(deps Deps) http.HandlerFunc`; `var bankSenders []string` (địa chỉ MB/TPBank được chấp nhận); `func isKnownBankSender(from string) bool`.

- [ ] **Step 1: Viết test thất bại**

Tạo `internal/handlers/inbox_webhook_test.go`:

```go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/handlers"
	"expensetracker/internal/inbound"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// postInbox signs body with secret and posts it to /inbox/{token}.
func postInbox(t *testing.T, router http.Handler, secret, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/inbox/"+token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(inbound.SignatureHeader, inbound.Sign(secret, body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func inboxPayload(from, to, subject, messageID, text string) []byte {
	b, _ := json.Marshal(inbound.Payload{
		From: from, To: to, Subject: subject, MessageID: messageID, Text: text,
	})
	return b
}

// enableInbox gives the user an inbox token and returns it.
func enableInbox(t *testing.T, deps handlers.Deps, userID int64) string {
	t.Helper()
	token, err := inbound.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	err = deps.Queries.SetInboxToken(context.Background(), sqlcgen.SetInboxTokenParams{
		ID: userID, InboxToken: pgtype.Text{String: token, Valid: true},
	})
	if err != nil {
		t.Fatalf("SetInboxToken() error = %v", err)
	}
	return token
}

func TestInboxWebhookStoresAForwardedBankEmail(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mb.com.vn", token+"@in.example.site",
		"Bien dong so du", "<m1@mail>", "TK 123 +50,000 VND")
	rec := postInbox(t, router, "s3cret", token, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /inbox = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	n, err := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("CountBankEmailsForUser() error = %v", err)
	}
	if n != 1 {
		t.Errorf("stored emails = %d, want 1", n)
	}
}

func TestInboxWebhookRejectsABadSignature(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mb.com.vn", token+"@in.example.site", "s", "<m2@mail>", "x")
	rec := postInbox(t, router, "the-wrong-secret", token, body)

	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /inbox with bad signature = %d, want %d", rec.Code, http.StatusForbidden)
	}
	n, _ := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if n != 0 {
		t.Errorf("stored emails = %d, want 0", n)
	}
}

func TestInboxWebhookRejectsAnUnknownToken(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)

	body := inboxPayload("no-reply@mb.com.vn", "nobody@in.example.site", "s", "<m3@mail>", "x")
	rec := postInbox(t, router, "s3cret", "nobody", body)

	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /inbox with unknown token = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestInboxWebhookStoresAStrangerAsIgnoredRatherThanDroppingIt(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("someone@example.com", token+"@in.example.site", "hi", "<m4@mail>", "hello")
	rec := postInbox(t, router, "s3cret", token, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /inbox from a stranger = %d, want %d", rec.Code, http.StatusOK)
	}
	rows, err := deps.Queries.ListRecentFailedBankEmails(context.Background(),
		sqlcgen.ListRecentFailedBankEmailsParams{UserID: userID, Limit: 10})
	if err != nil {
		t.Fatalf("ListRecentFailedBankEmails() error = %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("failed emails = %d, want 0 -- a stranger is ignored, not failed", len(rows))
	}
	n, _ := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if n != 1 {
		t.Errorf("stored emails = %d, want 1 -- an ignored email is still stored", n)
	}
}

func TestInboxWebhookStoresTheSameMessageOnlyOnce(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mb.com.vn", token+"@in.example.site", "s", "<dup@mail>", "x")
	for i := 0; i < 2; i++ {
		if rec := postInbox(t, router, "s3cret", token, body); rec.Code != http.StatusOK {
			t.Fatalf("POST #%d = %d, want %d", i+1, rec.Code, http.StatusOK)
		}
	}

	n, _ := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if n != 1 {
		t.Errorf("stored emails after two identical posts = %d, want 1", n)
	}
}

func TestInboxWebhookRejectsEveryRequestWhenNoSecretIsConfigured(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = ""
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mb.com.vn", token+"@in.example.site", "s", "<m5@mail>", "x")
	rec := postInbox(t, router, "", token, body)

	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /inbox with no configured secret = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
```

> `registerTestUser` phải trả về `int64` user id. Nếu helper hiện có trong `internal/handlers` chưa có dạng này, thêm nó vào `auth_handlers_test.go` cạnh `newTestDeps`, dùng lại đường đăng ký sẵn có rồi `GetUserByEmail` để lấy id.

- [ ] **Step 2: Chạy test để chắc nó fail**

Run: `TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./internal/handlers/ -run TestInboxWebhook -v`
Expected: FAIL — route `/inbox/{token}` chưa tồn tại, mọi request trả 404/405.

- [ ] **Step 3: Viết handler**

Tạo `internal/handlers/inbox_webhook.go`:

```go
package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"expensetracker/internal/inbound"
	"expensetracker/internal/sqlcgen"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// bankSenders are the addresses a balance-change notice may legitimately
// arrive from. This is the third auth layer and the load-bearing one:
// without it, anyone who learns a forwarding address could post invented
// email and put invented transactions straight into someone's books.
var bankSenders = []string{
	"@mb.com.vn",
	"@tpb.com.vn",
	"@tpbank.com.vn",
}

// isKnownBankSender reports whether from belongs to a bank this app reads.
// Matching is on the domain suffix rather than the whole address, because
// banks send from several local parts and change them without notice.
func isKnownBankSender(from string) bool {
	addr := strings.ToLower(strings.TrimSpace(from))
	if i := strings.LastIndex(addr, "<"); i >= 0 {
		addr = strings.TrimSuffix(addr[i+1:], ">")
	}
	for _, suffix := range bankSenders {
		if strings.HasSuffix(addr, suffix) {
			return true
		}
	}
	return false
}

// inboxWebhookHandler receives one forwarded email from the Cloudflare Email
// Worker, stores it raw, and answers 200 without doing anything else. Parsing
// it into a transaction is a later slice's job and deliberately not this
// request's: the Worker is waiting on this response, and a parser that is
// still wrong should not be able to lose an email.
//
// Three layers decide whether the caller is who they claim to be: the token
// in the path names an account, the HMAC proves the body came from our own
// Worker, and the sender address proves the message came from a bank. An
// email failing only the third is still stored, as 'ignored' -- keeping it
// costs nothing and makes "why did nothing happen" answerable.
func inboxWebhookHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4*inbound.MaxBodyBytes))
		if err != nil {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}

		if !inbound.Verify(deps.InboundWebhookSecret, body, r.Header.Get(inbound.SignatureHeader)) {
			http.Error(w, "bad signature", http.StatusForbidden)
			return
		}

		user, err := deps.Queries.GetUserByInboxToken(r.Context(), pgText(chi.URLParam(r, "token")))
		if errors.Is(err, pgx.ErrNoRows) {
			// No account owns this address -- most likely one whose owner
			// turned tracking off and regenerated their token.
			http.Error(w, "unknown inbox", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("inbox webhook: load user: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		payload, err := inbound.ParsePayload(body)
		if err != nil {
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}

		messageID := strings.TrimSpace(payload.MessageID)
		if messageID == "" {
			// Not every forwarder sets Message-ID, and the unique index that
			// stops a duplicate still needs a value.
			messageID = inbound.Fingerprint(payload)
		}

		status, reason := "pending", ""
		if !isKnownBankSender(payload.From) {
			status, reason = "ignored", "sender is not a bank this app reads"
		}

		_, err = deps.Queries.CreateBankEmail(r.Context(), sqlcgen.CreateBankEmailParams{
			UserID:        user.ID,
			MessageID:     messageID,
			FromAddress:   payload.From,
			Subject:       payload.Subject,
			Body:          inbound.TruncateBody(payload.Text),
			Status:        status,
			FailureReason: reason,
		})
		// pgx.ErrNoRows here is the ON CONFLICT DO NOTHING firing: the same
		// message already arrived once. That is a success, not a failure --
		// answering anything else would make the Worker retry forever.
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("inbox webhook: store email: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
```

- [ ] **Step 4: Thêm helper `pgText` nếu chưa có**

Kiểm tra: `grep -n 'func pgText' internal/handlers/`. Nếu chưa có, thêm vào `internal/handlers/req_params.go` cạnh `pgInt64`:

```go
// pgText wraps a string as the nullable text sqlc generates for a nullable
// column. An empty string is still a valid, non-NULL value here.
func pgText(v string) pgtype.Text {
	return pgtype.Text{String: v, Valid: true}
}
```

- [ ] **Step 5: Đăng ký route và miễn CSRF**

Trong `internal/handlers/app_router.go`, đổi dòng `r.Use(csrf.Middleware(deps.SecureCookies))` thành:

```go
	// The Worker webhook is the one route with no browser behind it, so it
	// has no cookie to double-submit and csrf.Middleware would reject every
	// email with 403. The exemption is expressed here, in the router, rather
	// than inside internal/csrf, which has no business knowing route paths.
	r.Use(csrfExcept(csrf.Middleware(deps.SecureCookies), "/inbox/"))
```

Thêm vào cuối `app_router.go`:

```go
// csrfExcept applies mw to every request except those whose path starts with
// prefix.
func csrfExcept(mw func(http.Handler) http.Handler, prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		guarded := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
			guarded.ServeHTTP(w, r)
		})
	}
}
```

Thêm `"strings"` vào import, và đăng ký route ngay sau `/healthz`:

```go
	// Public: the caller is the Cloudflare Email Worker, not a browser
	// carrying a session cookie.
	r.Post("/inbox/{token}", inboxWebhookHandler(deps))
```

- [ ] **Step 6: Chạy test**

Run: `TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./internal/handlers/ -run TestInboxWebhook -v`
Expected: PASS cả 6 test.

- [ ] **Step 7: Chạy toàn bộ suite để chắc không vỡ CSRF chỗ khác**

Run: `TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
gofmt -l . && go vet ./...
git add internal/handlers
git commit -m "feat: accept forwarded bank email at a signed, CSRF-exempt inbox webhook"
```

---

### Task 5: `deleteAccount` dọn cả `bank_emails`

**Files:**
- Modify: `internal/handlers/settings_handlers.go` (hàm `deleteAccount`)
- Test: `internal/handlers/settings_handlers_test.go`

**Interfaces:**
- Consumes: `Queries.DeleteBankEmailsForUser`.
- Produces: không có ký hiệu mới.

- [ ] **Step 1: Viết test thất bại**

Thêm vào `internal/handlers/settings_handlers_test.go`:

```go
func TestDeleteAccountRemovesStoredBankEmails(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	body := inboxPayload("no-reply@mb.com.vn", token+"@in.example.site", "s", "<del@mail>", "x")
	if rec := postInbox(t, router, "s3cret", token, body); rec.Code != http.StatusOK {
		t.Fatalf("seed email: POST /inbox = %d, want 200", rec.Code)
	}

	if err := handlers.DeleteAccountForTest(context.Background(), deps, userID); err != nil {
		t.Fatalf("deleteAccount() error = %v, want nil", err)
	}

	n, err := deps.Queries.CountBankEmailsForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("CountBankEmailsForUser() error = %v", err)
	}
	if n != 0 {
		t.Errorf("bank_emails after account delete = %d, want 0", n)
	}
}
```

> `deleteAccount` chưa export. Nếu suite hiện tại đã có đường gọi nó qua HTTP (`POST /settings/delete` với mật khẩu đúng), **dùng đường đó thay vì thêm `DeleteAccountForTest`** — đọc `settings_handlers_test.go` trước và bám theo cách đã có. Chỉ thêm shim khi thật sự không có đường nào.

- [ ] **Step 2: Chạy test để chắc nó fail**

Run: `TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./internal/handlers/ -run TestDeleteAccountRemovesStoredBankEmails -v`
Expected: FAIL — `bank_emails` còn 1 dòng, hoặc lỗi khoá ngoại.

- [ ] **Step 3: Thêm vào `deleteAccount`**

Trong `internal/handlers/settings_handlers.go`, chèn **sau** `DeleteTransactionsForUser` và **trước** `DeletePersonalCategoriesForUser`:

```go
	// After transactions and before the user row: transactions.bank_email_id
	// references bank_emails, so the rows pointing at these have to be gone
	// first. Spelled out for the same reason the rest of this list is --
	// leaving it to the cascade on users works only by accident of ordering.
	if err := qtx.DeleteBankEmailsForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete bank emails: %w", err)
	}
```

- [ ] **Step 4: Chạy test**

Run: `TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./internal/handlers/ -run TestDeleteAccount -v`
Expected: PASS, kể cả test xoá tài khoản đã có từ trước.

- [ ] **Step 5: Commit**

```bash
gofmt -l . && go vet ./...
git add internal/handlers
git commit -m "fix: delete an account's stored bank email along with the rest of its rows"
```

---

### Task 6: Thẻ "Email tracking" trên `/settings`

**Files:**
- Create: `internal/handlers/settings_inbox.go`
- Create: `internal/web/templates/settings_inbox.html`
- Test: `internal/handlers/settings_inbox_test.go`
- Modify: `internal/handlers/settings_handlers.go` (`settingsData`, `savedMessages`)
- Modify: `internal/handlers/app_router.go`
- Modify: `internal/web/web.go`
- Modify: `internal/web/templates/settings.html`

**Interfaces:**
- Consumes: `inbound.NewToken`, `Queries.SetInboxToken`, `Queries.ListRecentFailedBankEmails`, `Queries.RequeueFailedBankEmails`.
- Produces: `func enableInboxHandler(deps Deps) http.HandlerFunc`, `func disableInboxHandler(deps Deps) http.HandlerFunc`, `func retryFailedEmailsHandler(deps Deps) http.HandlerFunc`, `type failedEmailView struct{ Subject, ReceivedAt, Reason string }`, `const failedEmailsShown = 10`.

- [ ] **Step 1: Viết test thất bại**

Tạo `internal/handlers/settings_inbox_test.go`:

```go
package handlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

func TestSettingsHidesTheInboxCardWhenNoDomainIsConfigured(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = ""
	router := handlers.NewRouter(deps)
	registerTestUser(t, router, deps)

	body := getSettingsPage(t, router, deps)
	if strings.Contains(body, "Email tracking") {
		t.Error("settings page shows the Email tracking card with no INBOUND_DOMAIN set, want it hidden")
	}
}

func TestEnablingInboxShowsTheForwardingAddress(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)

	postSettingsForm(t, router, deps, "/settings/inbox/enable", nil)

	user, err := deps.Queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if !user.InboxToken.Valid || user.InboxToken.String == "" {
		t.Fatal("inbox_token after enable = NULL, want a token")
	}
	if body := getSettingsPage(t, router, deps); !strings.Contains(body, user.InboxToken.String+"@in.example.site") {
		t.Error("settings page does not show the forwarding address after enabling")
	}
}

func TestEnablingTwiceRotatesTheAddress(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)

	postSettingsForm(t, router, deps, "/settings/inbox/enable", nil)
	first, _ := deps.Queries.GetUserByID(context.Background(), userID)
	postSettingsForm(t, router, deps, "/settings/inbox/enable", nil)
	second, _ := deps.Queries.GetUserByID(context.Background(), userID)

	if first.InboxToken.String == second.InboxToken.String {
		t.Errorf("token after regenerate = %q, want a different one", second.InboxToken.String)
	}
}

func TestDisablingInboxClearsTheToken(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)

	postSettingsForm(t, router, deps, "/settings/inbox/enable", nil)
	postSettingsForm(t, router, deps, "/settings/inbox/disable", nil)

	user, _ := deps.Queries.GetUserByID(context.Background(), userID)
	if user.InboxToken.Valid {
		t.Errorf("inbox_token after disable = %q, want NULL", user.InboxToken.String)
	}
}

func TestRetryPutsFailedEmailsBackToPending(t *testing.T) {
	deps := newTestDeps(t)
	deps.InboundDomain = "in.example.site"
	deps.InboundWebhookSecret = "s3cret"
	router := handlers.NewRouter(deps)
	userID := registerTestUser(t, router, deps)
	token := enableInbox(t, deps, userID)

	markFailedBankEmail(t, deps, userID, token)

	postSettingsForm(t, router, deps, "/settings/inbox/retry", nil)

	rows, err := deps.Queries.ListRecentFailedBankEmails(context.Background(),
		sqlcgen.ListRecentFailedBankEmailsParams{UserID: userID, Limit: 10})
	if err != nil {
		t.Fatalf("ListRecentFailedBankEmails() error = %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("failed emails after retry = %d, want 0", len(rows))
	}
}
```

> Ba helper `getSettingsPage`, `postSettingsForm`, `markFailedBankEmail` phải được viết cùng lúc trong file này: hai cái đầu dựng request đã đăng nhập + đính CSRF theo đúng cách `withCSRF`/`csrfTokenFor` đã dùng; `markFailedBankEmail` chèn một dòng `bank_emails` với `status='failed'` bằng `deps.DB.Exec`. Đọc `settings_handlers_test.go` để lấy đúng cách tạo session cookie đang dùng.

- [ ] **Step 2: Chạy test để chắc nó fail**

Run: `TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./internal/handlers/ -run 'TestSettingsHides|TestEnablingInbox|TestEnablingTwice|TestDisablingInbox|TestRetryPuts' -v`
Expected: FAIL — route `/settings/inbox/enable` chưa có.

- [ ] **Step 3: Viết handler**

Tạo `internal/handlers/settings_inbox.go`:

```go
package handlers

import (
	"log"
	"net/http"

	"expensetracker/internal/auth"
	"expensetracker/internal/format"
	"expensetracker/internal/inbound"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// failedEmailsShown caps the "could not be read" list on the settings card.
// It is a prompt to fix the parser, not an archive: the rest stay in the
// table and the retry button covers all of them regardless of what is shown.
const failedEmailsShown = 10

// failedEmailView is one row of that list.
type failedEmailView struct {
	Subject    string
	ReceivedAt string
	Reason     string
}

// enableInboxHandler switches email tracking on, and switches it to a new
// address if it was already on. Regenerating is the revocation path: an
// address that leaked is retired the moment a new one is issued, because the
// webhook resolves the token against this column and nothing else.
func enableInboxHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		token, err := inbound.NewToken()
		if err != nil {
			log.Printf("enable inbox: new token: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := deps.Queries.SetInboxToken(r.Context(), sqlcgen.SetInboxTokenParams{
			ID: userID, InboxToken: pgtype.Text{String: token, Valid: true},
		}); err != nil {
			log.Printf("enable inbox: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/settings?saved=inbox-enabled", http.StatusSeeOther)
	}
}

// disableInboxHandler switches email tracking off. Stored email is left
// alone: it is the owner's own history, and turning the address off is not a
// request to erase what already arrived.
func disableInboxHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		if err := deps.Queries.SetInboxToken(r.Context(), sqlcgen.SetInboxTokenParams{
			ID: userID, InboxToken: pgtype.Text{},
		}); err != nil {
			log.Printf("disable inbox: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/settings?saved=inbox-disabled", http.StatusSeeOther)
	}
}

// retryFailedEmailsHandler puts every failed email back in the queue. This
// button is the entire reason email is stored raw before it is read: a parser
// fix is worth nothing if the messages it would now understand are gone.
func retryFailedEmailsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		if err := deps.Queries.RequeueFailedBankEmails(r.Context(), userID); err != nil {
			log.Printf("retry failed emails: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/settings?saved=inbox-retried", http.StatusSeeOther)
	}
}

// inboxSettingsData adds the Email tracking card's values to the settings
// page data. It returns nothing at all when no domain is configured, which is
// what makes the card disappear rather than render an address nobody can send
// to.
func inboxSettingsData(r *http.Request, deps Deps, userID int64, token pgtype.Text) (map[string]any, error) {
	if deps.InboundDomain == "" {
		return map[string]any{"InboxAvailable": false}, nil
	}

	data := map[string]any{
		"InboxAvailable": true,
		"InboxEnabled":   token.Valid && token.String != "",
	}
	if token.Valid && token.String != "" {
		data["InboxAddress"] = token.String + "@" + deps.InboundDomain
	}

	rows, err := deps.Queries.ListRecentFailedBankEmails(r.Context(), sqlcgen.ListRecentFailedBankEmailsParams{
		UserID: userID, Limit: failedEmailsShown,
	})
	if err != nil {
		return nil, err
	}
	views := make([]failedEmailView, 0, len(rows))
	for _, row := range rows {
		views = append(views, failedEmailView{
			Subject:    row.Subject,
			ReceivedAt: format.Timestamp(row.ReceivedAt, vietnamLocation),
			Reason:     row.FailureReason,
		})
	}
	data["InboxFailed"] = views
	return data, nil
}
```

- [ ] **Step 4: Nối vào `settingsData` và `savedMessages`**

Trong `internal/handlers/settings_handlers.go`, thêm vào map `savedMessages`:

```go
	"inbox-enabled":  "Email tracking is on. Forward your bank email to the address below.",
	"inbox-disabled": "Email tracking is off. The old address no longer accepts mail.",
	"inbox-retried":  "Those emails are queued to be read again.",
```

Và ở cuối `settingsData`, trước `return`, gộp dữ liệu thẻ mới vào map trả về:

```go
	inboxData, err := inboxSettingsData(r, deps, userID, user.InboxToken)
	if err != nil {
		return nil, err
	}
	for k, v := range inboxData {
		data[k] = v
	}
```

(đổi `return map[string]any{...}, nil` hiện tại thành gán vào biến `data` rồi `return data, nil`).

- [ ] **Step 5: Đăng ký ba route**

Trong `internal/handlers/app_router.go`, trong nhóm `RequireAuth`, cạnh các route settings khác:

```go
		pr.Post("/settings/inbox/enable", enableInboxHandler(deps))
		pr.Post("/settings/inbox/disable", disableInboxHandler(deps))
		pr.Post("/settings/inbox/retry", retryFailedEmailsHandler(deps))
```

- [ ] **Step 6: Viết template**

Tạo `internal/web/templates/settings_inbox.html`:

```html
{{define "settings_inbox"}}
{{if .InboxAvailable}}
<div class="bg-surface border border-border-card rounded-xl p-[18px] flex flex-col gap-[15px]">
  <div class="flex flex-col gap-1">
    <h2 class="text-base font-semibold text-ink">Email tracking</h2>
    <p class="text-sm text-ink-muted">Forward your bank's balance-change email here and $pend records the transaction for you.</p>
  </div>

  {{if .InboxEnabled}}
    <div class="flex flex-col gap-2">
      <span class="text-sm text-ink-muted">Your forwarding address</span>
      <div class="flex items-center gap-2 flex-wrap">
        <code data-copy-value="{{.InboxAddress}}" class="text-sm text-ink bg-surface-sunken border border-border-card rounded-lg px-3 py-2 break-all">{{.InboxAddress}}</code>
        <button type="button" data-copy-trigger class="text-sm text-accent px-3 py-2 rounded-lg border border-border-card">Copy</button>
      </div>
      <p class="text-sm text-ink-muted">In Gmail: Settings, then Forwarding, add this address, then create a filter for your bank's sender that forwards to it.</p>
    </div>

    <div class="flex items-center gap-2 flex-wrap">
      <form method="POST" action="/settings/inbox/enable">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <button type="submit" class="text-sm text-ink px-3 py-2 rounded-lg border border-border-card">Generate a new address</button>
      </form>
      <form method="POST" action="/settings/inbox/disable">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <button type="submit" class="text-sm text-expense px-3 py-2 rounded-lg border border-border-card">Turn off</button>
      </form>
    </div>

    {{if .InboxFailed}}
    <div class="flex flex-col gap-2 border-t border-border-card pt-[15px]">
      <span class="text-sm font-medium text-ink">Could not be read</span>
      <ul class="flex flex-col gap-2">
        {{range .InboxFailed}}
        <li class="flex flex-col">
          <span class="text-sm text-ink">{{.Subject}}</span>
          <span class="text-xs text-ink-muted">{{.ReceivedAt}} — {{.Reason}}</span>
        </li>
        {{end}}
      </ul>
      <form method="POST" action="/settings/inbox/retry">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <button type="submit" class="text-sm text-accent px-3 py-2 rounded-lg border border-border-card">Try these again</button>
      </form>
    </div>
    {{end}}
  {{else}}
    <form method="POST" action="/settings/inbox/enable">
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
      <button type="submit" class="text-sm text-white bg-accent px-3 py-2 rounded-lg">Turn on email tracking</button>
    </form>
  {{end}}
</div>
{{end}}
{{end}}
```

> Kiểm tra tên token màu (`text-ink`, `text-ink-muted`, `bg-surface-sunken`, `text-expense`, `bg-accent`) có thật trong `static/tailwind-config.js`; cái nào không có thì thay bằng token đã tồn tại. **Không** thêm hex, không `rgba(`.

- [ ] **Step 7: Nối template vào page set và trang settings**

Trong `internal/web/web.go`, đổi dòng `settings`:

```go
	"settings":        {"settings.html", "settings_inbox.html"},
```

Trong `internal/web/templates/settings.html`, chèn `{{template "settings_inbox" .}}` ngay trước thẻ danger zone (thẻ chứa `action="/settings/delete"`).

- [ ] **Step 8: Chạy test**

Run: `TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./internal/handlers/ -v`
Expected: PASS, kể cả `view_layout_test.go`.

- [ ] **Step 9: Commit**

```bash
gofmt -l . && go vet ./...
git add internal/handlers internal/web
git commit -m "feat: add the Email tracking settings card with its address, retry list and off switch"
```

---

### Task 7: Worker thật — ký HMAC và POST sang app

**Files:**
- Modify: `emailworker/src/index.js`
- Modify: `emailworker/wrangler.toml`
- Test: `emailworker/test/sign.test.mjs` (chạy bằng `node --test`)

**Interfaces:**
- Consumes: hợp đồng của `internal/inbound` — `Payload` field names, `Sign` (hex HMAC-SHA256 trên raw body), header `X-Inbox-Signature`.
- Produces: không có ký hiệu Go nào.

- [ ] **Step 1: Viết test thất bại cho hàm ký**

Tạo `emailworker/test/sign.test.mjs`:

```js
import { test } from "node:test";
import assert from "node:assert/strict";
import { sign } from "../src/sign.js";

// Fixture doi chieu voi Go: cung secret, cung body thi phai ra cung chuoi hex.
// Sinh lai bang:
//   go run ./cmd/... hoac mot test Go goi inbound.Sign("s3cret", []byte("{}"))
test("sign matches the Go side for a known fixture", async () => {
  const got = await sign("s3cret", "{}");
  assert.equal(got.length, 64, "HMAC-SHA256 hex must be 64 chars");
  assert.match(got, /^[0-9a-f]+$/);
});

test("sign is stable and secret-dependent", async () => {
  const a = await sign("k", "body");
  const b = await sign("k", "body");
  const c = await sign("other", "body");
  assert.equal(a, b);
  assert.notEqual(a, c);
});
```

- [ ] **Step 2: Chạy test để chắc nó fail**

Run: `cd emailworker && node --test test/`
Expected: FAIL — `../src/sign.js` không tồn tại.

- [ ] **Step 3: Viết `emailworker/src/sign.js`**

```js
// Ky HMAC-SHA256 tren raw body, tra ve hex thuong.
//
// Phai khop tung byte voi inbound.Sign trong Go (internal/inbound). Doi mot
// ben ma quen ben kia thi email dung lai im lang: app tra 403 va khong ai
// nhin thay, vi Worker khong co cho nao bao loi.
export async function sign(secret, body) {
  const enc = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    enc.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const mac = await crypto.subtle.sign("HMAC", key, enc.encode(body));
  return [...new Uint8Array(mac)]
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
```

- [ ] **Step 4: Chạy test**

Run: `cd emailworker && node --test test/`
Expected: PASS.

- [ ] **Step 5: Kiểm tra chéo với Go**

Chạy trong repo root:

```bash
cat > /tmp/sigcheck_test.go <<'EOF'
package inbound_test

import (
	"fmt"
	"testing"

	"expensetracker/internal/inbound"
)

func TestPrintFixtureSignature(t *testing.T) {
	fmt.Println("GO:", inbound.Sign("s3cret", []byte("{}")))
}
EOF
cp /tmp/sigcheck_test.go internal/inbound/sigcheck_test.go
go test ./internal/inbound/ -run TestPrintFixtureSignature -v | grep GO:
rm internal/inbound/sigcheck_test.go
cd emailworker && node -e 'import("./src/sign.js").then(m=>m.sign("s3cret","{}")).then(s=>console.log("JS:",s))'
```

Expected: hai chuỗi **giống hệt nhau**. Khác nhau là hợp đồng đã gãy — dừng lại và sửa trước khi đi tiếp.

- [ ] **Step 6: Viết handler email thật**

Thay toàn bộ `emailworker/src/index.js`:

```js
import { sign } from "./sign.js";

// Doc toi da 64KB phan text/plain. Thu ngan hang chi vai KB; cap nay de mot
// thu qua kho khong lam day database free tier.
const MAX_BODY = 64 * 1024;

async function readText(message) {
  const raw = await new Response(message.raw).text();
  // Phan than bat dau sau dong trong dau tien; khong parse MIME day du vi
  // thu ngan hang la text/plain mot phan.
  const split = raw.indexOf("\r\n\r\n");
  const body = split >= 0 ? raw.slice(split + 4) : raw;
  return body.slice(0, MAX_BODY);
}

export default {
  async email(message, env, ctx) {
    if (!env.INBOUND_WEBHOOK_SECRET || !env.APP_BASE_URL) {
      console.log("ERROR: worker is missing INBOUND_WEBHOOK_SECRET or APP_BASE_URL");
      return;
    }

    // Local part cua dia chi nhan la token dinh danh tai khoan.
    const token = String(message.to || "").split("@")[0].toLowerCase();
    if (!token) {
      console.log("ERROR: no token in recipient", message.to);
      return;
    }

    const payload = JSON.stringify({
      from: message.from,
      to: message.to,
      subject: message.headers.get("subject") || "",
      messageId: message.headers.get("message-id") || "",
      text: await readText(message),
    });

    const res = await fetch(
      `${env.APP_BASE_URL}/inbox/${encodeURIComponent(token)}`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Inbox-Signature": await sign(env.INBOUND_WEBHOOK_SECRET, payload),
        },
        body: payload,
      },
    );

    // Khong throw: nem loi o day khien Cloudflare bounce thu ve nguoi gui,
    // ma nguoi gui la ngan hang. Log lai la du -- thu van con trong hop thu
    // Gmail cua chu tai khoan de forward lai.
    if (!res.ok) {
      console.log("ERROR: app rejected the email", res.status, await res.text());
    }
  },
};
```

- [ ] **Step 7: Bỏ đoạn "bản tạm" trong `wrangler.toml`**

Xoá hai dòng nói đây là bản chỉ ghi log, giữ nguyên cảnh báo deploy tay.

- [ ] **Step 8: Đặt secret và deploy**

```bash
cd emailworker
export CLOUDFLARE_API_TOKEN=$(tr -d '\n\r ' < ~/.cf-api-token)
npx wrangler secret put INBOUND_WEBHOOK_SECRET   # dán đúng giá trị đã đặt cho app
npx wrangler secret put APP_BASE_URL             # https://<app>.onrender.com
npx wrangler deploy
```

- [ ] **Step 9: Commit**

```bash
cd /home/minhquan/projects/go/expense_tracker
git add emailworker
git commit -m "feat: make the Email Worker sign each forwarded message and post it to the inbox webhook"
```

---

### Task 8: Tài liệu — tiền tố `inbox_`, README, và ghi chú vận hành

**Files:**
- Modify: `CLAUDE.md` (bảng tiền tố, mục Architecture)
- Modify: `README.md`

**Interfaces:** không có.

- [ ] **Step 1: Thêm hai dòng vào bảng tiền tố trong `CLAUDE.md`**

Trong bảng "Finding a file in `internal/handlers`", chèn theo thứ tự bảng chữ cái:

```
| `inbox_` | the Worker webhook that receives forwarded bank email |
```

- [ ] **Step 2: Thêm một đoạn vào mục Architecture của `CLAUDE.md`**

Viết một dòng dài (file này để prose ở dòng dài, không xuống hàng thủ công), đặt ngay trước đoạn **Deployment**:

```
**Email ingestion** (`internal/inbound`, `internal/handlers/inbox_webhook.go`, `emailworker/`): a Cloudflare Email Worker receives mail forwarded to `<token>@in.<domain>`, signs the JSON body with HMAC-SHA256 under `INBOUND_WEBHOOK_SECRET`, and POSTs it to `/inbox/{token}` — a public route, and the one route exempt from CSRF, because the caller is the Worker rather than a browser with a cookie to double-submit. Three layers decide whether to trust it: the token names an account, the signature proves the body came from our own Worker, and the sender domain proves the mail came from a bank; a message failing only the last is stored as `ignored` rather than dropped, so "why did nothing happen" stays answerable. The handler stores the raw message and answers 200 without reading it — parsing lives in a later slice, and a parser that is still wrong must never be able to lose an email. `internal/inbound` holds every rule the Go side shares with the Worker (payload field names, the signature, the token); change one there and `emailworker/src/index.js` changes in the same commit, or email silently stops arriving.
```

- [ ] **Step 3: Thêm mục vào `README.md`**

Ở phần biến môi trường, thêm `INBOUND_DOMAIN` và `INBOUND_WEBHOOK_SECRET` với cùng cách mô tả các biến khác, kèm một câu: Worker deploy riêng bằng `npx wrangler deploy` từ `emailworker/`.

- [ ] **Step 4: Kiểm tra lần cuối và commit**

Run: `gofmt -l . && go vet ./... && TEST_DATABASE_URL="$TEST_DATABASE_URL" go test ./...`
Expected: PASS toàn bộ, `gofmt -l .` không in gì.

```bash
git add CLAUDE.md README.md
git commit -m "docs: write down the email ingestion path and the new inbox_ handler prefix"
```

---

## Kiểm thử đầu cuối thủ công (sau Task 8)

Không phải test tự động — đây là bước xác nhận đường thật, làm một lần:

1. Đặt `INBOUND_DOMAIN=in.ttth-caothang.site` và `INBOUND_WEBHOOK_SECRET=<giá trị>` trên Render, deploy.
2. Vào `/settings`, bật Email tracking, copy địa chỉ.
3. Trong Gmail: Settings → Forwarding → thêm địa chỉ đó. Gmail gửi một mã xác nhận tới chính địa chỉ ấy, và mã đó sẽ chạy vào Worker chứ không vào hộp thư nào — đọc mã bằng `npx wrangler tail` từ `emailworker/`.
4. Tạo filter Gmail: `from:(mb.com.vn OR tpb.com.vn)` → Forward to địa chỉ trên.
5. Chờ một email biến động số dư thật, rồi kiểm tra `bank_emails` có dòng `pending`.
6. **Ghi lại nội dung email thật đó** (che tên và số tài khoản) — đó là nguyên liệu bảng test của lát 2.

---

## Self-Review

**Spec coverage:** mục 5 (schema) → Task 1; mục 9 (cấu hình) → Task 3; mục 8 (webhook, ba lớp, miễn CSRF, adapter tách riêng) → Task 2 + Task 4; mục 5 `deleteAccount` → Task 5; mục 10 phần 1 (thẻ settings) → Task 6; Worker → Task 7; mục 11 (test) → rải trong Task 2/4/5/6.

**Cố ý để lại cho lát sau:** mục 6 `internal/bankmail`, mục 7 `internal/classify`, `category_hints`/`000015`, goroutine xử lý `pending`, nhãn `auto` và dấu "có thể trùng" (mục 10 phần 2), việc học từ thao tác sửa category (mục 10 phần 3). Không cái nào cần cho "forward email vào và nhìn thấy nó".

**Chệch khỏi spec, có chủ ý:** spec vẽ adapter payload là `internal/handlers/inbox_payload.go`; plan đặt nó ở `internal/inbound` vì nó không cần request lẫn database, đúng luật "helper không cần request/DB thì ra khỏi `internal/handlers`" trong CLAUDE.md, và vì Worker phải chia đúng luật ký với nó.

**Điểm rủi ro cao nhất:** hợp đồng chữ ký Go ↔ JS. Task 7 Step 5 là bước đối chiếu chéo bắt buộc, đừng bỏ qua.
