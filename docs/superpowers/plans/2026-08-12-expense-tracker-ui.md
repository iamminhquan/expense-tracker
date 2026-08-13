# Expense Tracker UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the high-fidelity UI design from `docs/design/design_handoff_expense_tracker/SPEC.md` on top of the existing working backend (auth, categories, transactions, dashboard), replacing the current bare-bones Tailwind markup with the full design system: tokens, nav, htmx-driven fragment mutations, CSRF protection, inline edit/delete, month filtering, empty states, and a real two-chart Tổng quan screen.

**Source of truth:** `docs/design/design_handoff_expense_tracker/SPEC.md` (referenced below as "SPEC.md §N"). `docs/design/design_handoff_expense_tracker/PROMPT.md` is the instructions the human gave for how this plan should be built; `Expense Tracker UI.dc.html` is a visual-reference-only mockup — never copy markup from it.

**Architecture (unchanged from the existing backend):** Go, `chi` router, `sqlc`-generated queries over `pgx/v5`, server-rendered `html/template` (one `*template.Template` per page name, each built from `layout.html` + exactly one page file, to keep `{{define "content"}}` from colliding across pages), cookie sessions, `golang-migrate`. This plan adds: htmx-driven fragment responses for every mutation, a stateless double-submit-cookie CSRF package, Tailwind CDN config extension (no build step added), and Chart.js (already CDN-pinned at `@4.4.4`).

**Tech added in this plan:** none beyond what's already in `go.mod` — no new Go modules, no new CDN scripts besides Google Fonts. `frontend-design` plugin was installed but is not itself a code dependency.

---

## Global Constraints

These apply to every task below; each task calls out only the parts of these that are newly *introduced* by that task.

- **Currency:** VND only, stored/passed as `int64` đồng, never float. Display via the new `vnd`/`vndSigned`/`vndBalance` template funcs — magnitude thousands-dot-separated + trailing `₫` (e.g. `50.000₫`), sign rules per SPEC.md §1/§3.3/§5.
- **Dates:** `dateFull` → `11/08/2026` (forms, desktop transaction date column), `dateShort` → `11/08` (transaction rows).
- **Ownership scoping:** every query touching `categories`/`transactions` filters by the authenticated `user_id` (already true in the existing queries); every new query/handler added by this plan preserves that.
- **CSRF:** every mutating request (`POST`/`PATCH`/`PUT`/`DELETE`) must carry a valid CSRF token, checked via `internal/csrf` (new package, Task 1). htmx requests get the token automatically via a global `htmx:configRequest` listener reading a `<meta name="csrf-token">` tag; the one plain (non-htmx) form — logout — carries it as a hidden `csrf_token` field, which the middleware also accepts as a fallback to the `X-CSRF-Token` header.
- **Fragments, not full reloads:** every mutation (create/edit/delete/color-patch/tab-switch/month-filter) responds with an HTML fragment for htmx to swap, never JSON, never (for htmx-triggered requests) a `3xx` redirect. The two exceptions, both required by SPEC.md/PROMPT.md: login/register success uses the `HX-Redirect` response header (not a body swap, since the destination is a different page entirely), and logout is a plain form POST that does a normal `http.Redirect` (it's explicitly not an htmx interaction).
- **No new build step or dependency without asking.** Tailwind theme customization uses the inline `tailwind.config` script (Play CDN JIT), not a build step. If a task's design implies needing something not already in the stack, stop and ask rather than adding it silently.
- **No icon library.** Dropdown carets, plus signs, etc. are text characters (`▾`, `＋`) or inline SVG, never an icon font/library.
- **Single accent color** (`#3B6ECF`) for primary actions/active states. Category colors (the 8-swatch palette + reserved gray) appear only as 8–10px dots and chart segments, never as backgrounds of buttons/badges/etc.
- **Breakpoint:** one breakpoint, `768px` (Tailwind's `md:`). Below it: bottom nav, single column, bottom sheets. At/above it: top nav, content capped at `880px` centered.
- **Default categories — exactly 9, exact values** (SPEC.md §4.3): expense `Ăn uống #D97757`, `Đi lại #5B8DEF`, `Mua sắm #8B7BD8`, `Hóa đơn #6BA292`, `Giải trí #E0A82E`, `Sức khỏe #D97AA0`, `Khác #A1A1AA`; income `Lương #4FA871`, `Thưởng #7CA65C`. This replaces the current 8-category seed from migration `000005`.
- **8-swatch color picker + reserved gray:** the category add/color-change UI offers exactly the 8 non-gray swatches from SPEC.md §1 (`#D97757 #5B8DEF #8B7BD8 #6BA292 #E0A82E #D97AA0 #4FA871 #7CA65C`). `#A1A1AA` is reserved for the system "Khác" category and the chart's synthetic "Khác" (other-aggregation) segment; it is never offered in the picker.
- **Category rename is name-only.** SPEC.md's "Sửa" action on a custom category only ever needs to change its name in every mockup/description — this plan does not add changing a category's `type` (expense↔income), since retyping a category with existing transactions of the old type creates ambiguity SPEC.md never addresses. If the human wants type-changing later, that's a separate follow-up.
- **Default-category color changes are allowed and shared.** SPEC.md §4.1 explicitly gives default categories a working "Đổi màu" action with no "only for custom categories" carve-out. This intentionally mutates a value every user sees (defaults are `user_id IS NULL`, shared) — documented here so it isn't mistaken for a bug during review.
- **Category delete semantics** (resolves a SPEC.md ambiguity the human already decided — see below): default categories can never be renamed or deleted, enforced at the handler (403), not just hidden in the UI. Custom **expense** categories with existing transactions get those transactions reassigned to the expense "Khác" default, then deleted, in one DB transaction. Custom **income** categories with existing transactions are **blocked from deletion** (409) rather than reassigned, because SPEC.md's 9-category total has no income-side "Khác" to reassign into — confirmed with the human via `AskUserQuestion` on 2026-08-12. Any custom category with zero transactions deletes directly.
- **Transaction validation** (closes gaps found while reading `transaction_handlers.go`): `type` must match the selected category's `type` (currently unchecked); `description` ≤ 200 runes (SPEC.md §8); `occurred_on` may not be more than **7 days** in the future — SPEC.md §8 says "không ở tương lai xa" (not far in the future) without a number, so 7 days is a documented, easily-adjustable default rather than a blocked-on-asking decision.
- **Mobile category edit affordance is a visible tap target, not true long-press.** SPEC.md describes "nhấn giữ mở action sheet" (press-and-hold); this plan implements a small always-visible "⋯" button per row instead, for reachability/accessibility and because JS press-and-hold gesture handling adds real complexity for no visual-token difference. Documented here rather than asked, since it doesn't change any spec'd color/spacing/type value.
- **CSRF test handshake pattern (introduced Task 1, reused by every later task's tests):** any test that issues a mutating `httptest.NewRequest` must first obtain a token via the `csrfTokenFor(t, router)` helper and attach it via `withCSRF(req, tok)` (both defined in Task 1, Step 8). Every task below that touches an existing POST/PATCH/DELETE test call site assumes this pattern is already in place from Task 1.

---

## File Structure (additions/changes only — see the existing repo for everything else)

```
expense_tracker/
├── cmd/server/main.go                              (modify: .Funcs(), template map changes)
├── internal/
│   ├── csrf/
│   │   ├── csrf.go                                 (new)
│   │   └── csrf_test.go                             (new)
│   ├── database/
│   │   ├── migrations/
│   │   │   ├── 000006_redesign_categories.up.sql   (new)
│   │   │   └── 000006_redesign_categories.down.sql (new)
│   │   └── queries/
│   │       ├── categories.sql                       (modify: new queries)
│   │       └── transactions.sql                      (modify: new queries)
│   ├── handlers/
│   │   ├── render.go                                (rewrite)
│   │   ├── format.go                                (new)
│   │   ├── format_test.go                           (new)
│   │   ├── router.go                                (modify: CSRF middleware, new routes)
│   │   ├── auth_handlers.go                          (rewrite)
│   │   ├── auth_handlers_test.go                      (modify)
│   │   ├── category_handlers.go                      (rewrite)
│   │   ├── category_handlers_test.go                  (modify)
│   │   ├── transaction_handlers.go                    (rewrite)
│   │   ├── transaction_handlers_test.go                (modify)
│   │   ├── report_handlers.go                        (rewrite)
│   │   ├── report_handlers_test.go                    (modify)
│   │   └── smoke_test.go                            (modify)
│   └── web/templates/
│       ├── layout.html                              (rewrite)
│       ├── auth.html                                (new, replaces login.html/register.html)
│       ├── auth_card_body.html                      (new)
│       ├── categories.html                          (rewrite)
│       ├── category_row.html                        (new fragment)
│       ├── transactions.html                        (rewrite)
│       ├── transaction_row.html                      (new fragment)
│       ├── dashboard.html                            (rewrite)
│       ├── login.html                                 (deleted, Task 2)
│       └── register.html                             (deleted, Task 2)
```

---

### Task 1: Nền tảng — CSRF, format helpers, render() redesign, layout/nav rewrite

**Files:**
- Create: `internal/csrf/csrf.go`
- Create: `internal/csrf/csrf_test.go`
- Create: `internal/handlers/format.go`
- Create: `internal/handlers/format_test.go`
- Modify: `internal/handlers/render.go`
- Modify: `internal/handlers/router.go`
- Modify: `internal/handlers/category_handlers.go` (render call sites only)
- Modify: `internal/handlers/transaction_handlers.go` (render call sites only)
- Modify: `internal/handlers/report_handlers.go` (render call sites only)
- Modify: `internal/handlers/auth_handlers.go` (render call sites only — full rewrite happens in Task 2)
- Modify: `internal/web/templates/layout.html`
- Modify: `cmd/server/main.go`
- Modify: `internal/handlers/auth_handlers_test.go` (add CSRF test helpers, `.Funcs()`, fix POST calls)
- Modify: `internal/handlers/category_handlers_test.go` (fix POST calls)
- Modify: `internal/handlers/transaction_handlers_test.go` (fix POST calls)
- Modify: `internal/handlers/report_handlers_test.go` (fix POST calls)
- Modify: `internal/handlers/smoke_test.go` (fix POST calls)

**Interfaces:**
- `csrf.Middleware(secure bool) func(http.Handler) http.Handler`
- `csrf.TokenFromRequest(r *http.Request) string`
- `handlers.TemplateFuncs() template.FuncMap`
- `render(w http.ResponseWriter, r *http.Request, deps Deps, page string, active string, data map[string]any)`
- `renderNamed(w http.ResponseWriter, r *http.Request, deps Deps, page string, tmplName string, active string, data map[string]any)`

- [ ] **Step 1: Write `internal/csrf/csrf_test.go`**

```go
package csrf_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expensetracker/internal/csrf"
)

func TestMiddlewareIssuesCookieOnFirstRequest(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrf.CookieName && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a csrf_token cookie to be set on first request")
	}
}

func TestMiddlewareRejectsMutationWithoutToken(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/transactions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a POST with no CSRF cookie/header, got %d", rec.Code)
	}
}

func TestMiddlewareAcceptsMatchingHeaderToken(t *testing.T) {
	var innerCalled bool
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("expected a token from the first request")
	}

	postReq := httptest.NewRequest(http.MethodPost, "/transactions", nil)
	postReq.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: token})
	postReq.Header.Set(csrf.HeaderName, token)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK || !innerCalled {
		t.Fatalf("expected matching header token to be accepted, got %d", postRec.Code)
	}
}

func TestMiddlewareAcceptsMatchingFormToken(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			token = c.Value
		}
	}

	postReq := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader("csrf_token="+token))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: token})
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected a matching csrf_token form field to be accepted for a plain form POST, got %d", postRec.Code)
	}
}

func TestMiddlewareRejectsMismatchedToken(t *testing.T) {
	handler := csrf.Middleware(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			token = c.Value
		}
	}

	postReq := httptest.NewRequest(http.MethodPost, "/transactions", nil)
	postReq.AddCookie(&http.Cookie{Name: csrf.CookieName, Value: token})
	postReq.Header.Set(csrf.HeaderName, "not-the-real-token")
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a mismatched token, got %d", postRec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/csrf/...`
Expected: FAIL — package `internal/csrf` doesn't exist yet.

- [ ] **Step 3: Write `internal/csrf/csrf.go`**

```go
package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

const (
	CookieName = "csrf_token"
	HeaderName = "X-CSRF-Token"
	FormField  = "csrf_token"
)

// Middleware implements the stateless double-submit-cookie CSRF pattern: no
// server-side token storage is needed because the check only proves the
// request carries back a value the server itself set as a cookie on an
// earlier same-site request. It issues a token cookie on any request that
// doesn't already have one, and rejects state-changing requests whose
// submitted token doesn't match the cookie.
func Middleware(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := ensureCookie(w, r, secure)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			if isMutating(r.Method) {
				submitted := r.Header.Get(HeaderName)
				if submitted == "" {
					// Plain <form method="POST"> submissions (logout) can't
					// set a custom header, so fall back to a hidden field
					// carrying the same token.
					submitted = r.FormValue(FormField)
				}
				if submitted == "" || subtle.ConstantTimeCompare([]byte(submitted), []byte(token)) != 1 {
					http.Error(w, "invalid or missing CSRF token", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ensureCookie returns the request's current CSRF token, generating and
// setting a fresh one on both the response and the in-flight request if
// none exists yet. Patching r (not just w) means the very first page a
// visitor loads already has a valid token to embed via TokenFromRequest,
// instead of only becoming available starting from their second request.
func ensureCookie(w http.ResponseWriter, r *http.Request, secure bool) (string, error) {
	if cookie, err := r.Cookie(CookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	return token, nil
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

// TokenFromRequest reads the current request's CSRF token for embedding
// into a rendered page (meta tag, hidden form input). Middleware guarantees
// a token is present on the request by the time handlers run.
func TokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/csrf/...`
Expected: PASS

- [ ] **Step 5: Write `internal/handlers/format_test.go`**

```go
package handlers

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestVND(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0₫"},
		{999, "999₫"},
		{50000, "50.000₫"},
		{18500000, "18.500.000₫"},
		{-85000, "85.000₫"}, // vnd() shows magnitude only, never a sign
	}
	for _, tc := range cases {
		if got := vnd(tc.in); got != tc.want {
			t.Errorf("vnd(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVNDSigned(t *testing.T) {
	if got := vndSigned(85000, "expense"); got != "-85.000₫" {
		t.Errorf("vndSigned(85000, expense) = %q, want -85.000₫", got)
	}
	if got := vndSigned(18500000, "income"); got != "+18.500.000₫" {
		t.Errorf("vndSigned(18500000, income) = %q, want +18.500.000₫", got)
	}
}

func TestVNDBalance(t *testing.T) {
	if got := vndBalance(120000); got != "+120.000₫" {
		t.Errorf("vndBalance(120000) = %q, want +120.000₫", got)
	}
	if got := vndBalance(-45000); got != "-45.000₫" {
		t.Errorf("vndBalance(-45000) = %q, want -45.000₫", got)
	}
}

func TestDateFormatting(t *testing.T) {
	d := pgtype.Date{Time: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), Valid: true}
	if got := dateFull(d); got != "11/08/2026" {
		t.Errorf("dateFull = %q, want 11/08/2026", got)
	}
	if got := dateShort(d); got != "11/08" {
		t.Errorf("dateShort = %q, want 11/08", got)
	}
	if got := dateFull(pgtype.Date{}); got != "" {
		t.Errorf("dateFull(invalid) = %q, want empty string", got)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/handlers/... -run TestVND`
Expected: FAIL — `vnd`/`vndSigned`/`vndBalance`/`dateFull`/`dateShort` undefined.

- [ ] **Step 7: Write `internal/handlers/format.go`**

```go
package handlers

import (
	"html/template"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// TemplateFuncs returns the FuncMap every page template needs for
// formatting money and dates per SPEC.md section 1: thousands-dot-separated
// integers with a trailing ₫, and dd/mm/yyyy dates.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"vnd":        vnd,
		"vndSigned":  vndSigned,
		"vndBalance": vndBalance,
		"dateFull":   dateFull,
		"dateShort":  dateShort,
	}
}

// vnd formats the magnitude of n as thousands-dot-separated đồng, e.g.
// 50000 -> "50.000₫". The sign is never shown here; callers needing a sign
// use vndSigned (transaction rows) or vndBalance (a total that can itself
// be negative).
func vnd(n int64) string {
	if n < 0 {
		n = -n
	}
	return formatThousands(n) + "₫"
}

// vndSigned formats a transaction amount with the sign SPEC.md section 3.3
// assigns by transaction type: "-" for expense, "+" for anything else.
func vndSigned(n int64, txnType string) string {
	sign := "+"
	if txnType == "expense" {
		sign = "-"
	}
	return sign + vnd(n)
}

// vndBalance formats a total (e.g. "remaining this month") whose sign comes
// from the number's own sign, since unlike a single transaction it can
// itself be negative.
func vndBalance(n int64) string {
	sign := "+"
	if n < 0 {
		sign = "-"
	}
	return sign + vnd(n)
}

func formatThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ".")
}

// dateFull formats a DATE column as dd/mm/yyyy, e.g. "11/08/2026" -- used in
// forms and the desktop transaction list's date column.
func dateFull(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("02/01/2006")
}

// dateShort formats a DATE column as dd/mm, e.g. "11/08" -- used in the
// transaction list row and mobile card.
func dateShort(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("02/01")
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/handlers/... -run TestVND -run TestDateFormatting`
Expected: PASS

- [ ] **Step 9: Rewrite `internal/handlers/render.go`**

```go
package handlers

import (
	"bytes"
	"log"
	"net/http"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/csrf"
)

// render executes page's "layout" template. It always injects CSRFToken; if
// active is non-empty it also injects the ShowNav/ActiveNav/UserName/
// UserInitial fields layout.html's nav blocks need, by loading the
// authenticated user via deps.Queries. Pre-auth pages (login/register) pass
// active="" so nav data is skipped entirely and layout.html's
// {{if .ShowNav}} blocks correctly stay hidden -- a missing map key
// evaluates falsy in html/template's {{if}}, so no explicit false is
// needed.
func render(w http.ResponseWriter, r *http.Request, deps Deps, page string, active string, data map[string]any) {
	renderNamed(w, r, deps, page, "layout", active, data)
}

// renderNamed is render's more general form: it executes tmplName instead
// of always "layout", for fragment responses (a swapped-in row, a
// tab-switch card body, etc.) that must not re-render the full page shell.
func renderNamed(w http.ResponseWriter, r *http.Request, deps Deps, page string, tmplName string, active string, data map[string]any) {
	tmpl, ok := deps.Templates[page]
	if !ok {
		log.Printf("render: no template registered for page %q", page)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if data == nil {
		data = map[string]any{}
	}
	data["CSRFToken"] = csrf.TokenFromRequest(r)

	if active != "" {
		navData, err := authPageData(r, deps, active)
		if err != nil {
			log.Printf("render: load nav data: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		for k, v := range navData {
			data[k] = v
		}
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		log.Printf("render: execute template %q (block %q): %v", page, tmplName, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("render: write response for %q: %v", page, err)
	}
}

// authPageData loads the fields every authenticated page's nav needs: the
// user's display name/initial, and which nav link is active.
func authPageData(r *http.Request, deps Deps, active string) (map[string]any, error) {
	userID, _ := auth.UserIDFromContext(r.Context())
	user, err := deps.Queries.GetUserByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	initial := "?"
	if runes := []rune(user.Name); len(runes) > 0 {
		initial = strings.ToUpper(string(runes[0]))
	}
	return map[string]any{
		"ShowNav":     true,
		"ActiveNav":   active,
		"UserName":    user.Name,
		"UserInitial": initial,
	}, nil
}
```

- [ ] **Step 10: Update render call sites in `category_handlers.go`, `transaction_handlers.go`, `report_handlers.go`, `auth_handlers.go`**

These are mechanical signature fixes only (add `r` param, add `active` param) so the build compiles again; the feature rewrites for auth (Task 2), categories (Tasks 3–4), transactions (Tasks 5–9), and dashboard (Tasks 10–11) happen later.

In `internal/handlers/category_handlers.go`, both `render(w, deps, "categories", ...)` call sites (the GET/POST path in `categoriesPage` and the FK-violation path in `deleteCategoryHandler`) become:

```go
render(w, r, deps, "categories", "categories", map[string]any{"Categories": categories})
```

and

```go
render(w, r, deps, "categories", "categories", map[string]any{
	"Categories": categories,
	"Error":      "Không thể xóa danh mục đang được sử dụng bởi các giao dịch",
})
```

In `internal/handlers/transaction_handlers.go`, the single `render(w, deps, "transactions", ...)` call becomes:

```go
render(w, r, deps, "transactions", "transactions", map[string]any{
	"Transactions": transactions,
	"Categories":   categories,
	"Error":        formErr,
})
```

In `internal/handlers/report_handlers.go`, the single `render(w, deps, "dashboard", ...)` call becomes:

```go
render(w, r, deps, "dashboard", "dashboard", map[string]any{
	"TotalExpense":        totals.TotalExpense,
	"TotalIncome":         totals.TotalIncome,
	"BreakdownLabelsJSON": template.JS(labelsJSON),
	"BreakdownValuesJSON": template.JS(valuesJSON),
	"BreakdownColorsJSON": template.JS(colorsJSON),
})
```

In `internal/handlers/auth_handlers.go`, all five `render(w, deps, "register"/"login", ...)` call sites get `r` added and `active: ""` added (pre-auth pages never show nav), e.g.:

```go
render(w, r, deps, "register", "", map[string]any{})
```

```go
render(w, r, deps, "register", "", map[string]any{"Error": "Vui lòng nhập họ tên", "Name": name, "Email": email})
```

(same pattern for the other three register-page calls and the two login-page calls — insert `r` as the second argument and `""` as the fifth).

- [ ] **Step 11: Wire `.Funcs()` into every template map and add the CSRF middleware in `router.go`**

In `internal/handlers/router.go`, add the import `"expensetracker/internal/csrf"` and register the middleware before any routes, so it also covers pre-auth `POST /login`/`POST /register`:

```go
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(csrf.Middleware(deps.SecureCookies))
```

In `cmd/server/main.go`, change every `template.Must(template.ParseFiles(...))` entry to route Funcs through a named receiver template first (Funcs must be registered before Parse to be available during template execution):

```go
templates := map[string]*template.Template{
	"register":     template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/register.html")),
	"login":        template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/login.html")),
	"categories":   template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/categories.html")),
	"transactions": template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/transactions.html")),
	"dashboard":    template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/dashboard.html")),
}
```

(`template.New("layout.html")`'s name argument is never used for lookup — `ExecuteTemplate` looks up by each file's own `{{define "..."}}` name — it only exists so there's a receiver to call `.Funcs()` on before parsing.)

- [ ] **Step 12: Rewrite `internal/web/templates/layout.html`**

```html
{{define "layout"}}
<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Sổ chi tiêu</title>
  <meta name="csrf-token" content="{{.CSRFToken}}">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Be+Vietnam+Pro:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
  <script src="https://unpkg.com/htmx.org@1.9.12"></script>
  <script>
    document.addEventListener('htmx:configRequest', function (evt) {
      var meta = document.querySelector('meta[name="csrf-token"]');
      if (meta) {
        evt.detail.headers['X-CSRF-Token'] = meta.getAttribute('content');
      }
    });
  </script>
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    tailwind.config = {
      theme: {
        extend: {
          colors: {
            app: '#FAF9F7',
            surface: '#FFFFFF',
            'surface-alt': '#FCFBF9',
            track: '#F4F3F0',
            'border-card': '#E9E7E4',
            'border-input': '#E2E0DC',
            'border-list': '#F1EFEC',
            'border-nav': '#EDEBE7',
            ink: '#1B1A18',
            'ink-muted': '#57534E',
            'ink-faint': '#8A8781',
            'ink-faintest': '#9C9891',
            placeholder: '#A8A49D',
            'ink-zero': '#C6C2BB',
            accent: '#3B6ECF',
            expense: '#C2410C',
            income: '#2F7D5B',
          },
          fontFamily: {
            sans: ['"Be Vietnam Pro"', 'system-ui', 'sans-serif'],
            mono: ['"JetBrains Mono"', 'monospace'],
          },
        },
      },
    };
  </script>
  <style>
    .font-mono { font-variant-numeric: tabular-nums; }
  </style>
</head>
<body class="bg-app text-ink font-sans antialiased">
  {{if .ShowNav}}{{template "nav_desktop" .}}{{end}}
  <main class="{{if .ShowNav}}pb-24 md:pb-10{{end}}">
    {{template "content" .}}
  </main>
  {{if .ShowNav}}{{template "nav_mobile" .}}{{end}}
</body>
</html>
{{end}}

{{define "nav_desktop"}}
<nav class="hidden md:flex items-center h-[54px] bg-surface border-b border-border-nav sticky top-0 z-40 px-6 gap-[26px]">
  <a href="/dashboard" class="flex items-center gap-2 shrink-0">
    <span class="w-[18px] h-[18px] rounded-[5px] bg-accent block"></span>
    <span class="text-sm font-semibold text-ink">Sổ chi tiêu</span>
  </a>
  <div class="flex items-center gap-1">
    <a href="/dashboard" class="px-[11px] py-[6px] rounded-[7px] text-[13px] {{if eq .ActiveNav "dashboard"}}text-accent font-semibold bg-accent/10{{else}}text-[#6B6862] hover:bg-track hover:text-ink{{end}}">Tổng quan</a>
    <a href="/transactions" class="px-[11px] py-[6px] rounded-[7px] text-[13px] {{if eq .ActiveNav "transactions"}}text-accent font-semibold bg-accent/10{{else}}text-[#6B6862] hover:bg-track hover:text-ink{{end}}">Giao dịch</a>
    <a href="/categories" class="px-[11px] py-[6px] rounded-[7px] text-[13px] {{if eq .ActiveNav "categories"}}text-accent font-semibold bg-accent/10{{else}}text-[#6B6862] hover:bg-track hover:text-ink{{end}}">Danh mục</a>
  </div>
  <div class="flex items-center gap-2 ml-auto">
    <span class="w-6 h-6 rounded-full bg-border-nav flex items-center justify-center text-[11px] font-semibold text-ink-muted">{{.UserInitial}}</span>
    <span class="text-[13px] text-ink-muted">{{.UserName}}</span>
    <form method="POST" action="/logout">
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
      <button type="submit" class="text-[13px] text-ink-faint hover:text-ink">Đăng xuất</button>
    </form>
  </div>
</nav>
{{end}}

{{define "nav_mobile"}}
<nav class="md:hidden fixed bottom-0 inset-x-0 bg-surface border-t border-border-nav flex z-40" style="padding: 10px 0 22px 0;">
  <a href="/dashboard" class="flex-1 flex flex-col items-center justify-center gap-1 min-h-[56px]">
    <span class="w-[5px] h-[5px] rounded-full {{if eq .ActiveNav "dashboard"}}bg-accent{{else}}bg-transparent{{end}}"></span>
    <span class="text-[11px] {{if eq .ActiveNav "dashboard"}}text-accent font-semibold{{else}}text-ink-faint{{end}}">Tổng quan</span>
  </a>
  <a href="/transactions" class="flex-1 flex flex-col items-center justify-center gap-1 min-h-[56px]">
    <span class="w-[5px] h-[5px] rounded-full {{if eq .ActiveNav "transactions"}}bg-accent{{else}}bg-transparent{{end}}"></span>
    <span class="text-[11px] {{if eq .ActiveNav "transactions"}}text-accent font-semibold{{else}}text-ink-faint{{end}}">Giao dịch</span>
  </a>
  <a href="/categories" class="flex-1 flex flex-col items-center justify-center gap-1 min-h-[56px]">
    <span class="w-[5px] h-[5px] rounded-full {{if eq .ActiveNav "categories"}}bg-accent{{else}}bg-transparent{{end}}"></span>
    <span class="text-[11px] {{if eq .ActiveNav "categories"}}text-accent font-semibold{{else}}text-ink-faint{{end}}">Danh mục</span>
  </a>
  <button type="button" onclick="document.getElementById('logout-dialog').showModal()" class="flex-1 flex flex-col items-center justify-center gap-1 min-h-[56px]">
    <span class="w-[5px] h-[5px] rounded-full bg-transparent"></span>
    <span class="text-[11px] text-ink-faint">Đăng xuất</span>
  </button>
</nav>
<dialog id="logout-dialog" class="rounded-xl p-5 max-w-[300px] w-[85vw] backdrop:bg-black/30">
  <p class="text-[15px] font-semibold text-ink mb-4">Đăng xuất khỏi Sổ chi tiêu?</p>
  <div class="flex gap-2 justify-end">
    <button type="button" onclick="document.getElementById('logout-dialog').close()" class="px-4 h-9 rounded-lg border border-border-input text-[13px] text-ink">Hủy</button>
    <form method="POST" action="/logout">
      <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
      <button type="submit" class="px-4 h-9 rounded-lg bg-accent text-white text-[13px] font-semibold">Đăng xuất</button>
    </form>
  </div>
</dialog>
{{end}}
```

- [ ] **Step 13: Add the shared CSRF test helpers to `internal/handlers/auth_handlers_test.go`**

Add near the top of the file (after `newTestDeps`), and add `.Funcs(handlers.TemplateFuncs())` to `newTestDeps`'s template map the same way Step 11 did for `main.go`:

```go
templates := map[string]*template.Template{
	"register":     template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/register.html")),
	"login":        template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/login.html")),
	"categories":   template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/categories.html")),
	"transactions": template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/transactions.html")),
	"dashboard":    template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/dashboard.html")),
}
```

```go
// csrfTokenFor issues a GET request to obtain a fresh csrf_token cookie.
// Every mutating request built by a test must attach this cookie's value as
// both the cookie itself and the X-CSRF-Token header via withCSRF, or
// csrf.Middleware (wired into every route in Step 11) rejects it with 403.
func csrfTokenFor(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrf.CookieName {
			return c
		}
	}
	t.Fatal("expected a csrf_token cookie to be set on GET /login")
	return nil
}

// withCSRF attaches the cookie and header a mutating request needs to pass
// csrf.Middleware.
func withCSRF(req *http.Request, tok *http.Cookie) {
	req.AddCookie(tok)
	req.Header.Set(csrf.HeaderName, tok.Value)
}
```

Add `"expensetracker/internal/csrf"` to the file's imports.

- [ ] **Step 14: Fix every existing mutating request in the handler test files**

Every `httptest.NewRequest(http.MethodPost, ...)` (there is no PATCH/DELETE yet) built anywhere in `auth_handlers_test.go`, `category_handlers_test.go`, `transaction_handlers_test.go`, `report_handlers_test.go`, and `smoke_test.go` needs a CSRF token attached before `router.ServeHTTP` is called on it. The pattern is the same everywhere — obtain a token once per test (or once inside a shared helper like `loginAndGetCookie`), then call `withCSRF` on each request right after building it:

```go
tok := csrfTokenFor(t, router)
req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
withCSRF(req, tok)
rec := httptest.NewRecorder()
router.ServeHTTP(rec, req)
```

Apply this to every POST request in:
- `auth_handlers_test.go`: `TestRegisterThenLoginFlow` (both the register and the login POST), `TestRegisterValidatesInput` (the POST inside the subtest loop), `TestRegisterDuplicateEmailShowsSpecificMessage` (both POSTs), `TestSessionCookieAttributes` (the register POST and the logout POST — logout must use the **form-field** fallback from Step 3 since it's a real, non-htmx form: build it as `strings.NewReader("csrf_token=" + tok.Value)` with `Content-Type: application/x-www-form-urlencoded` instead of calling `withCSRF`, matching how the rewritten `nav_desktop`/`nav_mobile` logout forms will actually submit in Task 2 onward).
- `category_handlers_test.go`: inside the shared `loginAndGetCookie` helper (its `/register` POST — also return nothing new, just fix internally), `TestCreateAndListCategories`'s create POST, `TestDeleteCategoryInUseShowsFriendlyError`'s delete POST.
- `transaction_handlers_test.go`: `TestTransactionCRUDAndIsolation`'s create POST and delete POST, `TestCreateTransactionRejectsForeignCategory`'s forged create POST.
- `report_handlers_test.go`: `TestDashboardShowsMonthlyTotal`'s create POST.
- `smoke_test.go`: `TestEndToEndRegisterAddTransactionSeeDashboard`'s create POST (register already goes through the now-fixed `loginAndGetCookie`).

- [ ] **Step 15: Run the full test suite to verify it passes**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS. This confirms CSRF, the format helpers, and the render() redesign are all correctly wired before any visual work begins.

---

### Task 2: Auth — unified tabbed `auth.html`, htmx tab switch, `HX-Redirect` on success

**Files:**
- Create: `internal/web/templates/auth.html`
- Create: `internal/web/templates/auth_card_body.html`
- Delete: `internal/web/templates/login.html`
- Delete: `internal/web/templates/register.html`
- Rewrite: `internal/handlers/auth_handlers.go`
- Modify: `cmd/server/main.go` (template map: replace `register`/`login` keys with one `auth` key)
- Modify: `internal/handlers/auth_handlers_test.go`

**Interfaces:**
- `loginPage(deps Deps) http.HandlerFunc`, `registerPage(deps Deps) http.HandlerFunc` — same signatures as before, internals rewritten.
- New response contract: on successful login/register, handlers set the `HX-Redirect: /transactions` response header and return `200` with no body, instead of `http.Redirect` with `303`. On validation failure, they re-render only the `auth_card_body` fragment (200).
- GET `/login`/`/register`: if the request carries `HX-Request: true` (htmx tab switch), respond with just the `auth_card_body` fragment; otherwise respond with the full `auth` page (direct navigation, refresh, bookmark).

Design notes (SPEC.md §2): logo/app-name/tagline are centered above the card on desktop and replaced by a left-aligned page title ("Đăng nhập"/"Đăng ký") + left-aligned logo on mobile — both are always in the DOM, toggled with `hidden md:flex` / `md:hidden` rather than server-side branching, since the breakpoint is a pure CSS concern. The segmented tab control, fields, submit button, and footer link all live inside `#auth-card`, which is what htmx swaps on tab switch — this keeps the outer logo/title from flashing during the swap. The "Quên mật khẩu?" text is rendered as a plain `<span>`, not a link, because a working forgot-password flow is out of scope (PROMPT.md forbids unrequested features) and a dead `<a href="#">` would be worse than an inert label with identical styling.

- [ ] **Step 1: Update `TestLoginAndRegisterPagesRenderDistinctContent` and add new auth tests to `internal/handlers/auth_handlers_test.go`**

Replace the body of `TestLoginAndRegisterPagesRenderDistinctContent` (the `action="/login"` / `action="/register"` checks no longer apply — both forms now submit via `hx-post`) with:

```go
func TestLoginAndRegisterPagesRenderDistinctContent(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	loginReq := httptest.NewRequest(http.MethodGet, "/login", nil)
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	loginBody := loginRec.Body.String()
	if !strings.Contains(loginBody, `hx-post="/login"`) {
		t.Fatalf("expected GET /login body to contain a form posting to /login via htmx, got: %s", loginBody)
	}
	if strings.Contains(loginBody, `name="name"`) {
		t.Fatalf("expected GET /login body to NOT contain the register-only name field, got: %s", loginBody)
	}

	registerReq := httptest.NewRequest(http.MethodGet, "/register", nil)
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)

	registerBody := registerRec.Body.String()
	if !strings.Contains(registerBody, `hx-post="/register"`) {
		t.Fatalf("expected GET /register body to contain a form posting to /register via htmx, got: %s", registerBody)
	}
	if !strings.Contains(registerBody, `name="name"`) {
		t.Fatalf("expected GET /register body to contain the name field, got: %s", registerBody)
	}
}

// TestAuthTabSwitchReturnsFragmentOnly covers the htmx tab-switch contract
// from SPEC.md section 2: a request carrying HX-Request must get back just
// the auth_card_body fragment, not a full <html> page, so htmx can swap it
// into #auth-card without a flash of the logo/tagline re-rendering.
func TestAuthTabSwitchReturnsFragmentOnly(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<html") {
		t.Fatalf("expected an htmx fragment response with no <html> wrapper, got: %s", body)
	}
	if !strings.Contains(body, `hx-post="/register"`) {
		t.Fatalf("expected the register form fragment, got: %s", body)
	}
}

// TestLoginSuccessSendsHXRedirect covers PROMPT.md's "every mutation
// returns a fragment, never a full reload" rule applied to the one case
// that needs a real navigation: login/register success signal it via the
// HX-Redirect response header instead of an HTTP 3xx, since htmx treats a
// 3xx as just another response to swap in, not a page navigation.
func TestLoginSuccessSendsHXRedirect(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "hx-redirect@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	tok := csrfTokenFor(t, router)
	regForm := url.Values{"name": {"HX"}, "email": {email}, "password": {"s3cret-pass"}, "password_confirm": {"s3cret-pass"}}
	regReq := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(regForm.Encode()))
	regReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(regReq, tok)
	router.ServeHTTP(httptest.NewRecorder(), regReq)

	tok2 := csrfTokenFor(t, router)
	loginForm := url.Values{"email": {email}, "password": {"s3cret-pass"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(loginForm.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(loginReq, tok2)
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with HX-Redirect on success, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	if got := loginRec.Header().Get("HX-Redirect"); got != "/transactions" {
		t.Fatalf("expected HX-Redirect: /transactions, got %q", got)
	}
}

// TestRegisterPasswordMismatchShowsError covers the new "Nhập lại mật khẩu"
// field SPEC.md section 2 adds to the register tab.
func TestRegisterPasswordMismatchShowsError(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	email := "mismatch@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Mismatch"}, "email": {email}, "password": {"s3cret-pass"}, "password_confirm": {"different"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering the form, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Mật khẩu nhập lại không khớp") {
		t.Fatalf("expected password-mismatch error, got: %s", rec.Body.String())
	}
	if _, err := deps.Queries.GetUserByEmail(context.Background(), email); err == nil {
		t.Fatal("expected no user to be created when passwords don't match")
	}
}
```

Also update `TestRegisterThenLoginFlow`: both the register POST and the login POST now succeed with `http.StatusOK` + `HX-Redirect: /transactions` instead of `http.StatusSeeOther` — change both `if rec.Code != http.StatusSeeOther` / `if loginRec.Code != http.StatusSeeOther` checks accordingly (status `http.StatusOK`, and optionally assert the header too).

Also update `TestRegisterDuplicateEmailShowsSpecificMessage`: the first registration's success check changes from `StatusSeeOther` to `StatusOK`; the duplicate-email message assertion changes from `"Email đã được sử dụng"` to SPEC.md's exact copy `"Email này đã được dùng."`.

Also update `newTestDeps`'s template map: replace the `"register"`/`"login"` entries with:

```go
"auth": template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("../web/templates/layout.html", "../web/templates/auth.html", "../web/templates/auth_card_body.html")),
```

- [ ] **Step 2: Run the new/changed tests to verify they fail**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run 'TestLogin|TestRegister|TestAuthTabSwitch'`
Expected: FAIL — compile error (`auth_card_body` template missing, handlers still redirect instead of setting `HX-Redirect`).

- [ ] **Step 3: Write `internal/web/templates/auth_card_body.html`**

```html
{{define "auth_card_body"}}
<div class="flex bg-track rounded-lg p-[3px] gap-1">
  <button type="button"
    hx-get="/login" hx-target="#auth-card" hx-swap="innerHTML" hx-push-url="true"
    class="flex-1 h-8 rounded-[6px] text-[13px] {{if eq .Tab "login"}}bg-white text-ink font-semibold shadow-[0_1px_2px_rgba(0,0,0,0.06)]{{else}}text-ink-muted{{end}}">Đăng nhập</button>
  <button type="button"
    hx-get="/register" hx-target="#auth-card" hx-swap="innerHTML" hx-push-url="true"
    class="flex-1 h-8 rounded-[6px] text-[13px] {{if eq .Tab "register"}}bg-white text-ink font-semibold shadow-[0_1px_2px_rgba(0,0,0,0.06)]{{else}}text-ink-muted{{end}}">Đăng ký</button>
</div>

{{if eq .Tab "register"}}
<form hx-post="/register" hx-target="#auth-card" hx-swap="innerHTML" class="flex flex-col gap-4">
  <div class="flex flex-col gap-[6px]">
    <label class="text-[12px] font-medium text-ink-muted" for="name">Tên</label>
    <input id="name" name="name" type="text" value="{{.Name}}" required
      class="h-12 md:h-[38px] px-3 rounded-[10px] md:rounded-lg border border-border-input text-[15px] md:text-[13px] focus:outline-none focus:border-accent focus:ring-[3px] focus:ring-accent/[0.12]">
  </div>
  <div class="flex flex-col gap-[6px]">
    <label class="text-[12px] font-medium text-ink-muted" for="email">Email</label>
    <input id="email" name="email" type="email" value="{{.Email}}" placeholder="ban@email.com" required
      class="h-12 md:h-[38px] px-3 rounded-[10px] md:rounded-lg border border-border-input text-[15px] md:text-[13px] placeholder:text-placeholder focus:outline-none focus:border-accent focus:ring-[3px] focus:ring-accent/[0.12]">
  </div>
  <div class="flex flex-col gap-[6px]">
    <label class="text-[12px] font-medium text-ink-muted" for="password">Mật khẩu</label>
    <input id="password" name="password" type="password" required
      class="h-12 md:h-[38px] px-3 rounded-[10px] md:rounded-lg border border-border-input text-[15px] md:text-[13px] focus:outline-none focus:border-accent focus:ring-[3px] focus:ring-accent/[0.12]">
  </div>
  <div class="flex flex-col gap-[6px]">
    <label class="text-[12px] font-medium text-ink-muted" for="password_confirm">Nhập lại mật khẩu</label>
    <input id="password_confirm" name="password_confirm" type="password" required
      class="h-12 md:h-[38px] px-3 rounded-[10px] md:rounded-lg border border-border-input text-[15px] md:text-[13px] focus:outline-none focus:border-accent focus:ring-[3px] focus:ring-accent/[0.12]">
  </div>
  {{if .Error}}
  <p class="text-[12px] text-expense bg-[#FEF2F2] border border-[#FECACA] rounded-lg px-3 py-2">{{.Error}}</p>
  {{end}}
  <button type="submit" class="h-[50px] md:h-10 rounded-[10px] md:rounded-lg bg-accent text-white text-[13px] font-semibold">Tạo tài khoản</button>
</form>
<p class="text-center text-[12px] text-ink-faint">Đã có tài khoản?
  <button type="button" hx-get="/login" hx-target="#auth-card" hx-swap="innerHTML" hx-push-url="true" class="text-accent font-medium">Đăng nhập</button>
</p>
{{else}}
<form hx-post="/login" hx-target="#auth-card" hx-swap="innerHTML" class="flex flex-col gap-4">
  <div class="flex flex-col gap-[6px]">
    <label class="text-[12px] font-medium text-ink-muted" for="email">Email</label>
    <input id="email" name="email" type="email" value="{{.Email}}" placeholder="ban@email.com" required
      class="h-12 md:h-[38px] px-3 rounded-[10px] md:rounded-lg border border-border-input text-[15px] md:text-[13px] placeholder:text-placeholder focus:outline-none focus:border-accent focus:ring-[3px] focus:ring-accent/[0.12]">
  </div>
  <div class="flex flex-col gap-[6px]">
    <div class="flex items-baseline justify-between">
      <label class="text-[12px] font-medium text-ink-muted" for="password">Mật khẩu</label>
      <span class="text-[12px] text-accent">Quên mật khẩu?</span>
    </div>
    <input id="password" name="password" type="password" required
      class="h-12 md:h-[38px] px-3 rounded-[10px] md:rounded-lg border border-border-input text-[15px] md:text-[13px] focus:outline-none focus:border-accent focus:ring-[3px] focus:ring-accent/[0.12]">
  </div>
  {{if .Error}}
  <p class="text-[12px] text-expense bg-[#FEF2F2] border border-[#FECACA] rounded-lg px-3 py-2">{{.Error}}</p>
  {{end}}
  <button type="submit" class="h-[50px] md:h-10 rounded-[10px] md:rounded-lg bg-accent text-white text-[13px] font-semibold">Đăng nhập</button>
</form>
<p class="text-center text-[12px] text-ink-faint">Chưa có tài khoản?
  <button type="button" hx-get="/register" hx-target="#auth-card" hx-swap="innerHTML" hx-push-url="true" class="text-accent font-medium">Đăng ký</button>
</p>
{{end}}
{{end}}
```

- [ ] **Step 4: Write `internal/web/templates/auth.html`**

```html
{{define "content"}}
<div class="min-h-screen flex items-center justify-center bg-app px-5 py-12 md:py-[72px]">
  <div class="w-full max-w-[380px] flex flex-col gap-5">
    <div class="hidden md:flex flex-col items-center gap-2">
      <span class="w-[30px] h-[30px] rounded-[9px] bg-accent block"></span>
      <p class="text-[17px] font-semibold text-ink">Sổ chi tiêu</p>
      <p class="text-[13px] text-ink-faint">Ghi lại thu chi hằng ngày của bạn</p>
    </div>
    <div class="md:hidden flex flex-col gap-3">
      <span class="w-[34px] h-[34px] rounded-[9px] bg-accent block"></span>
      <p class="text-[22px] font-semibold text-ink">{{if eq .Tab "register"}}Đăng ký{{else}}Đăng nhập{{end}}</p>
    </div>
    <div id="auth-card" class="flex flex-col gap-4 md:border md:border-border-card md:rounded-xl md:p-[22px]">
      {{template "auth_card_body" .}}
    </div>
  </div>
</div>
{{end}}
```

- [ ] **Step 5: Delete `internal/web/templates/login.html` and `internal/web/templates/register.html`**

Both are fully superseded by `auth.html` + `auth_card_body.html`.

- [ ] **Step 6: Rewrite `internal/handlers/auth_handlers.go`**

```go
package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps holds shared dependencies for handlers.
//
// Templates is keyed by page name (e.g. "auth", "categories"). Each entry
// is a *template.Template built from layout.html plus that page's own
// template file(s), so that every template set has only a single
// {{define "content"}} block in scope. This avoids a collision that would
// occur if all page templates were parsed together into one shared
// *template.Template: Go's html/template registers "content" as a single
// global name per template set, so the last-parsed page's block would win
// for every page.
type Deps struct {
	DB         *pgxpool.Pool
	Queries    *sqlcgen.Queries
	Sessions   *auth.Manager
	Templates  map[string]*template.Template
	CookieName string
	// SecureCookies gates the Secure attribute on the session and CSRF
	// cookies; see internal/config.Config.SecureCookies for how it's
	// populated.
	SecureCookies bool
}

// renderAuthFragmentOrPage renders just the auth_card_body fragment when
// the request came from htmx (tab switch, or a validation re-render after
// a failed submit -- both submit forms with hx-post/hx-target="#auth-card"
// so a fragment is exactly what's expected back), or the full auth page
// shell on a direct navigation/refresh/bookmark.
func renderAuthFragmentOrPage(w http.ResponseWriter, r *http.Request, deps Deps, data map[string]any) {
	if r.Header.Get("HX-Request") == "true" {
		renderNamed(w, r, deps, "auth", "auth_card_body", "", data)
		return
	}
	render(w, r, deps, "auth", "", data)
}

func loginPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderAuthFragmentOrPage(w, r, deps, map[string]any{"Tab": "login"})
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := deps.Queries.GetUserByEmail(r.Context(), email)
		if err != nil || !auth.VerifyPassword(user.PasswordHash, password) {
			renderNamed(w, r, deps, "auth", "auth_card_body", "", map[string]any{
				"Tab": "login", "Error": "Email hoặc mật khẩu không đúng.", "Email": email,
			})
			return
		}

		startSession(w, r, deps, user.ID)
		w.Header().Set("HX-Redirect", "/transactions")
		w.WriteHeader(http.StatusOK)
	}
}

func registerPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderAuthFragmentOrPage(w, r, deps, map[string]any{"Tab": "register"})
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")
		passwordConfirm := r.FormValue("password_confirm")

		fail := func(msg string) {
			renderNamed(w, r, deps, "auth", "auth_card_body", "", map[string]any{
				"Tab": "register", "Error": msg, "Name": name, "Email": email,
			})
		}

		if name == "" {
			fail("Vui lòng nhập họ tên")
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			fail("Email không hợp lệ")
			return
		}
		if len([]rune(password)) < 8 {
			fail("Mật khẩu phải có ít nhất 8 ký tự.")
			return
		}
		if password != passwordConfirm {
			fail("Mật khẩu nhập lại không khớp.")
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			fail("Không thể tạo tài khoản")
			return
		}

		user, err := deps.Queries.CreateUser(r.Context(), sqlcgen.CreateUserParams{
			Email:        email,
			PasswordHash: hash,
			Name:         name,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				fail("Email này đã được dùng.")
				return
			}
			log.Printf("register: create user: %v", err)
			fail("Không thể tạo tài khoản, vui lòng thử lại")
			return
		}

		startSession(w, r, deps, user.ID)
		w.Header().Set("HX-Redirect", "/transactions")
		w.WriteHeader(http.StatusOK)
	}
}

func logoutHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(deps.CookieName); err == nil {
			deps.Sessions.DeleteSession(r.Context(), cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     deps.CookieName,
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Secure:   deps.SecureCookies,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func startSession(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	token, expiresAt, err := deps.Sessions.CreateSession(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     deps.CookieName,
		Value:    token,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   deps.SecureCookies,
	})
}
```

(`logoutHandler` and `startSession` are unchanged from before Task 1/2 other than already having gone through Task 1's mechanical fixes — reproduced here in full since this is a whole-file rewrite.)

- [ ] **Step 7: Update `cmd/server/main.go`'s template map**

Replace the `"register"`/`"login"` entries with a single `"auth"` entry parsing all three files:

```go
templates := map[string]*template.Template{
	"auth":         template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/auth.html", "internal/web/templates/auth_card_body.html")),
	"categories":   template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/categories.html")),
	"transactions": template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/transactions.html")),
	"dashboard":    template.Must(template.New("layout.html").Funcs(handlers.TemplateFuncs()).ParseFiles("internal/web/templates/layout.html", "internal/web/templates/dashboard.html")),
}
```

- [ ] **Step 8: Run the full test suite to verify it passes**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

- [ ] **Step 9: Manual smoke check**

Run: `go run ./cmd/server`, visit `http://localhost:8080/login` in a browser, confirm: tab switch between Đăng nhập/Đăng ký swaps the card without a full page reload (check Network tab — no full document request), submitting valid register credentials redirects to `/transactions`, submitting a wrong password shows the red error block and preserves the typed email.

---

### Task 3: Danh mục — schema migration and new sqlc queries

**Files:**
- Create: `internal/database/migrations/000006_redesign_categories.up.sql`
- Create: `internal/database/migrations/000006_redesign_categories.down.sql`
- Modify: `internal/database/queries/categories.sql`
- Modify: `internal/database/queries/transactions.sql`
- Modify: `internal/database/migrate_test.go`
- Modify (generated code): `internal/sqlcgen/categories.sql.go`, `internal/sqlcgen/transactions.sql.go`

**Interfaces:**
- `Queries.UpdateCategoryName(ctx, UpdateCategoryNameParams{ID, UserID, Name}) (Category, error)`
- `Queries.UpdateCategoryColor(ctx, UpdateCategoryColorParams{ID, UserID, Color}) (Category, error)`
- `Queries.GetDefaultCategoryForReassignment(ctx) (Category, error)`
- `Queries.ListCategoriesWithTransactionCounts(ctx, UserID pgtype.Int8) ([]ListCategoriesWithTransactionCountsRow, error)`
- `Queries.ReassignCategoryTransactions(ctx, ReassignCategoryTransactionsParams{CategoryID, CategoryID_2, UserID}) (int64, error)`
- `Queries.CountTransactionsForCategory(ctx, CountTransactionsForCategoryParams{CategoryID, UserID}) (int64, error)`
- `Queries.ListDistinctTransactionMonths(ctx, UserID int64) ([]pgtype.Date, error)`
- `Queries.MonthlyTotalsSeries(ctx, MonthlyTotalsSeriesParams{UserID, OccurredOn, OccurredOn_2}) ([]MonthlyTotalsSeriesRow, error)`

- [ ] **Step 1: Extend `internal/database/migrate_test.go`'s `TestMigrationsApplyCleanly` to assert the new seed/constraints**

Replace the body between `m.Up()` succeeding and `m.Down()` with:

```go
	var count int
	if err := db.QueryRow("SELECT count(*) FROM categories WHERE user_id IS NULL").Scan(&count); err != nil {
		t.Fatalf("query default categories: %v", err)
	}
	if count != 9 {
		t.Fatalf("expected 9 default categories, got %d", count)
	}

	var anUongColor string
	if err := db.QueryRow("SELECT color FROM categories WHERE user_id IS NULL AND name = 'Ăn uống'").Scan(&anUongColor); err != nil {
		t.Fatalf("query Ăn uống color: %v", err)
	}
	if anUongColor != "#D97757" {
		t.Fatalf("expected Ăn uống to be seeded with #D97757, got %q", anUongColor)
	}

	if _, err := db.Exec(
		`INSERT INTO categories (user_id, name, type, color) VALUES (NULL, 'Bad Color Test', 'expense', '#000000')`,
	); err == nil {
		t.Fatal("expected inserting a category with a color outside the fixed palette to violate the CHECK constraint")
	}

	var userID int64
	if err := db.QueryRow(
		`INSERT INTO users (email, password_hash, name) VALUES ('migrate-constraint-test@example.com', 'x', 'Constraint Test') RETURNING id`,
	).Scan(&userID); err != nil {
		t.Fatalf("insert throwaway user: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Dup', 'expense', '#D97757')`, userID,
	); err != nil {
		t.Fatalf("insert first Dup category: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO categories (user_id, name, type, color) VALUES ($1, 'Dup', 'expense', '#5B8DEF')`, userID,
	); err == nil {
		t.Fatal("expected a second category with the same (user_id, type, name) to violate the unique index")
	}

	if err := m.Down(); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/database/...`
Expected: FAIL — still only 8 default categories, no CHECK constraint, no unique index (migration `000006` doesn't exist yet).

- [ ] **Step 3: Write `internal/database/migrations/000006_redesign_categories.up.sql`**

```sql
-- Replace the placeholder 8-category seed from 000005 with the exact 9
-- categories and hex values SPEC.md section 4.3 specifies, and add the
-- constraints the redesigned category UI needs: a fixed 9-color palette (8
-- user-selectable swatches + the reserved gray for "Khác"), and a per-user
-- uniqueness rule so a user can't have two categories of the same type
-- sharing a name (Postgres treats NULL user_id rows as always-distinct in
-- a unique index, so this only constrains real per-user rows, not the
-- shared defaults).
DELETE FROM categories WHERE user_id IS NULL;

INSERT INTO categories (user_id, name, type, color) VALUES
    (NULL, 'Ăn uống', 'expense', '#D97757'),
    (NULL, 'Đi lại', 'expense', '#5B8DEF'),
    (NULL, 'Mua sắm', 'expense', '#8B7BD8'),
    (NULL, 'Hóa đơn', 'expense', '#6BA292'),
    (NULL, 'Giải trí', 'expense', '#E0A82E'),
    (NULL, 'Sức khỏe', 'expense', '#D97AA0'),
    (NULL, 'Khác', 'expense', '#A1A1AA'),
    (NULL, 'Lương', 'income', '#4FA871'),
    (NULL, 'Thưởng', 'income', '#7CA65C');

ALTER TABLE categories ADD CONSTRAINT categories_color_check CHECK (
    color IN ('#D97757', '#5B8DEF', '#8B7BD8', '#6BA292', '#E0A82E', '#D97AA0', '#4FA871', '#7CA65C', '#A1A1AA')
);

CREATE UNIQUE INDEX idx_categories_user_type_name ON categories (user_id, type, name);
```

- [ ] **Step 4: Write `internal/database/migrations/000006_redesign_categories.down.sql`**

```sql
DROP INDEX idx_categories_user_type_name;
ALTER TABLE categories DROP CONSTRAINT categories_color_check;

DELETE FROM categories WHERE user_id IS NULL;

INSERT INTO categories (user_id, name, type, color) VALUES
    (NULL, 'Ăn uống', 'expense', '#ef4444'),
    (NULL, 'Di chuyển', 'expense', '#f97316'),
    (NULL, 'Giải trí', 'expense', '#eab308'),
    (NULL, 'Hóa đơn', 'expense', '#8b5cf6'),
    (NULL, 'Sức khỏe', 'expense', '#ec4899'),
    (NULL, 'Mua sắm', 'expense', '#06b6d4'),
    (NULL, 'Lương', 'income', '#22c55e'),
    (NULL, 'Thu nhập khác', 'income', '#10b981');
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/database/...`
Expected: PASS.

- [ ] **Step 6: Add the new queries to `internal/database/queries/categories.sql`**

Append:

```sql
-- name: UpdateCategoryName :one
UPDATE categories SET name = $3
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: UpdateCategoryColor :one
UPDATE categories SET color = $3
WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
RETURNING *;

-- name: GetDefaultCategoryForReassignment :one
SELECT * FROM categories
WHERE user_id IS NULL AND type = 'expense' AND name = 'Khác';

-- name: ListCategoriesWithTransactionCounts :many
SELECT c.*, COUNT(t.id) AS transaction_count
FROM categories c
LEFT JOIN transactions t ON t.category_id = c.id AND t.user_id = $1
WHERE c.user_id = $1 OR c.user_id IS NULL
GROUP BY c.id
ORDER BY c.user_id NULLS FIRST, c.name;
```

- [ ] **Step 7: Add the new queries to `internal/database/queries/transactions.sql`**

Append:

```sql
-- name: ReassignCategoryTransactions :execrows
UPDATE transactions SET category_id = $1
WHERE category_id = $2 AND user_id = $3;

-- name: CountTransactionsForCategory :one
SELECT COUNT(*)::bigint AS count FROM transactions
WHERE category_id = $1 AND user_id = $2;

-- name: ListDistinctTransactionMonths :many
SELECT DISTINCT date_trunc('month', occurred_on)::date AS month
FROM transactions
WHERE user_id = $1
ORDER BY month DESC;

-- name: MonthlyTotalsSeries :many
SELECT
    date_trunc('month', occurred_on)::date AS month,
    COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)::bigint AS total_expense,
    COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0)::bigint AS total_income
FROM transactions
WHERE user_id = $1 AND occurred_on >= $2 AND occurred_on < $3
GROUP BY month
ORDER BY month;
```

- [ ] **Step 8: Regenerate `internal/sqlcgen`**

Run: `sqlc generate` from the repo root (uses `sqlc.yaml`, which already points `queries`/`schema`/`out` at the right directories).

If the `sqlc` CLI isn't available in the execution environment, hand-write the equivalent generated code below instead — it matches the exact conventions the existing `internal/sqlcgen/categories.sql.go` and `internal/sqlcgen/transactions.sql.go` already use (same `const <queryName> = ...` + `<QueryName>Params` + method pattern), so a later real `sqlc generate` run will reproduce it byte-for-byte and this is not a fork the codebase has to maintain by hand going forward.

Append to `internal/sqlcgen/categories.sql.go`:

```go
const updateCategoryName = `-- name: UpdateCategoryName :one
UPDATE categories SET name = $3
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, name, type, color, created_at
`

type UpdateCategoryNameParams struct {
	ID     int64       `json:"id"`
	UserID pgtype.Int8 `json:"user_id"`
	Name   string      `json:"name"`
}

func (q *Queries) UpdateCategoryName(ctx context.Context, arg UpdateCategoryNameParams) (Category, error) {
	row := q.db.QueryRow(ctx, updateCategoryName, arg.ID, arg.UserID, arg.Name)
	var i Category
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Type,
		&i.Color,
		&i.CreatedAt,
	)
	return i, err
}

const updateCategoryColor = `-- name: UpdateCategoryColor :one
UPDATE categories SET color = $3
WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
RETURNING id, user_id, name, type, color, created_at
`

type UpdateCategoryColorParams struct {
	ID     int64       `json:"id"`
	UserID pgtype.Int8 `json:"user_id"`
	Color  string      `json:"color"`
}

func (q *Queries) UpdateCategoryColor(ctx context.Context, arg UpdateCategoryColorParams) (Category, error) {
	row := q.db.QueryRow(ctx, updateCategoryColor, arg.ID, arg.UserID, arg.Color)
	var i Category
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Type,
		&i.Color,
		&i.CreatedAt,
	)
	return i, err
}

const getDefaultCategoryForReassignment = `-- name: GetDefaultCategoryForReassignment :one
SELECT id, user_id, name, type, color, created_at FROM categories
WHERE user_id IS NULL AND type = 'expense' AND name = 'Khác'
`

func (q *Queries) GetDefaultCategoryForReassignment(ctx context.Context) (Category, error) {
	row := q.db.QueryRow(ctx, getDefaultCategoryForReassignment)
	var i Category
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Type,
		&i.Color,
		&i.CreatedAt,
	)
	return i, err
}

const listCategoriesWithTransactionCounts = `-- name: ListCategoriesWithTransactionCounts :many
SELECT c.id, c.user_id, c.name, c.type, c.color, c.created_at, COUNT(t.id) AS transaction_count
FROM categories c
LEFT JOIN transactions t ON t.category_id = c.id AND t.user_id = $1
WHERE c.user_id = $1 OR c.user_id IS NULL
GROUP BY c.id
ORDER BY c.user_id NULLS FIRST, c.name
`

type ListCategoriesWithTransactionCountsRow struct {
	ID               int64              `json:"id"`
	UserID           pgtype.Int8        `json:"user_id"`
	Name             string             `json:"name"`
	Type             string             `json:"type"`
	Color            string             `json:"color"`
	CreatedAt        pgtype.Timestamptz `json:"created_at"`
	TransactionCount int64              `json:"transaction_count"`
}

func (q *Queries) ListCategoriesWithTransactionCounts(ctx context.Context, userID pgtype.Int8) ([]ListCategoriesWithTransactionCountsRow, error) {
	rows, err := q.db.Query(ctx, listCategoriesWithTransactionCounts, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListCategoriesWithTransactionCountsRow
	for rows.Next() {
		var i ListCategoriesWithTransactionCountsRow
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.Name,
			&i.Type,
			&i.Color,
			&i.CreatedAt,
			&i.TransactionCount,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
```

Append to `internal/sqlcgen/transactions.sql.go`:

```go
const reassignCategoryTransactions = `-- name: ReassignCategoryTransactions :execrows
UPDATE transactions SET category_id = $1
WHERE category_id = $2 AND user_id = $3
`

type ReassignCategoryTransactionsParams struct {
	CategoryID   int64 `json:"category_id"`
	CategoryID_2 int64 `json:"category_id_2"`
	UserID       int64 `json:"user_id"`
}

func (q *Queries) ReassignCategoryTransactions(ctx context.Context, arg ReassignCategoryTransactionsParams) (int64, error) {
	result, err := q.db.Exec(ctx, reassignCategoryTransactions, arg.CategoryID, arg.CategoryID_2, arg.UserID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const countTransactionsForCategory = `-- name: CountTransactionsForCategory :one
SELECT COUNT(*)::bigint AS count FROM transactions
WHERE category_id = $1 AND user_id = $2
`

type CountTransactionsForCategoryParams struct {
	CategoryID int64 `json:"category_id"`
	UserID     int64 `json:"user_id"`
}

func (q *Queries) CountTransactionsForCategory(ctx context.Context, arg CountTransactionsForCategoryParams) (int64, error) {
	row := q.db.QueryRow(ctx, countTransactionsForCategory, arg.CategoryID, arg.UserID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const listDistinctTransactionMonths = `-- name: ListDistinctTransactionMonths :many
SELECT DISTINCT date_trunc('month', occurred_on)::date AS month
FROM transactions
WHERE user_id = $1
ORDER BY month DESC
`

func (q *Queries) ListDistinctTransactionMonths(ctx context.Context, userID int64) ([]pgtype.Date, error) {
	rows, err := q.db.Query(ctx, listDistinctTransactionMonths, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []pgtype.Date
	for rows.Next() {
		var month pgtype.Date
		if err := rows.Scan(&month); err != nil {
			return nil, err
		}
		items = append(items, month)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const monthlyTotalsSeries = `-- name: MonthlyTotalsSeries :many
SELECT
    date_trunc('month', occurred_on)::date AS month,
    COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)::bigint AS total_expense,
    COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0)::bigint AS total_income
FROM transactions
WHERE user_id = $1 AND occurred_on >= $2 AND occurred_on < $3
GROUP BY month
ORDER BY month
`

type MonthlyTotalsSeriesParams struct {
	UserID       int64       `json:"user_id"`
	OccurredOn   pgtype.Date `json:"occurred_on"`
	OccurredOn_2 pgtype.Date `json:"occurred_on_2"`
}

type MonthlyTotalsSeriesRow struct {
	Month        pgtype.Date `json:"month"`
	TotalExpense int64       `json:"total_expense"`
	TotalIncome  int64       `json:"total_income"`
}

func (q *Queries) MonthlyTotalsSeries(ctx context.Context, arg MonthlyTotalsSeriesParams) ([]MonthlyTotalsSeriesRow, error) {
	rows, err := q.db.Query(ctx, monthlyTotalsSeries, arg.UserID, arg.OccurredOn, arg.OccurredOn_2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MonthlyTotalsSeriesRow
	for rows.Next() {
		var i MonthlyTotalsSeriesRow
		if err := rows.Scan(&i.Month, &i.TotalExpense, &i.TotalIncome); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
```

- [ ] **Step 9: Build to verify the generated code compiles**

Run: `go build ./...`
Expected: success (nothing calls the new queries yet, but they must compile standalone).

---

### Task 4: Danh mục — handlers, page, color popover, rename, delete-with-reassign

**Files:**
- Rewrite: `internal/handlers/category_handlers.go`
- Rewrite: `internal/web/templates/categories.html`
- Create: `internal/web/templates/category_row.html`
- Modify: `internal/handlers/format.go` (add `swatches` to `TemplateFuncs`)
- Modify: `internal/handlers/router.go` (new routes)
- Modify: `internal/database/queries/categories.sql` + `internal/sqlcgen/categories.sql.go` (one more query)
- Modify: `internal/handlers/category_handlers_test.go`

**Interfaces:**
- `categoriesPage(deps Deps) http.HandlerFunc` — GET renders the page, POST creates a category.
- `updateCategoryColorHandler`, `editCategoryHandler`, `viewCategoryRowHandler`, `updateCategoryNameHandler`, `deleteCategoryHandler` — all `func(deps Deps) http.HandlerFunc`.
- Routes: `PATCH /categories/{id}/color`, `GET /categories/{id}/edit`, `GET /categories/{id}/view`, `PATCH /categories/{id}/name`, `DELETE /categories/{id}` (replaces the old `POST /categories/{id}/delete`).

Design notes:
- **Every row the server ever sends back — whether from the initial page render, a color-patch, a rename, or a new-category create — goes through one `categoryRowData(...) map[string]any` helper**, never a raw `sqlcgen.Category`/`ListCategoriesWithTransactionCountsRow` struct passed straight to the template. This matters because `category_row.html` reads an optional `.OOBTarget` field: Go's `html/template` hard-errors ("can't evaluate field OOBTarget") if the root value is a struct that lacks the field, but silently returns empty (falsy) for a missing key on a `map[string]any`. Using a map uniformly sidesteps that entirely.
- **Create response uses an out-of-band swap, not a direct target.** The add-category form doesn't know at submit time whether the new row belongs in `#category-list-expense` or `#category-list-income` — that depends on which "Loại" the user picked. So the form itself uses `hx-swap="none"`, and the response is the new row with `hx-swap-oob="afterbegin:#category-list-{{.Type}}"` set directly on it (via a non-empty `OOBTarget`); htmx processes OOB-marked elements regardless of the triggering element's own `hx-swap`.
- **The add-category form is a single DOM node relocated between desktop sidebar and mobile bottom sheet**, not two independent copies. Duplicating it would mean two elements sharing one `id`, which breaks the `HX-Retarget: #add-category-form` header the server uses to redraw the form-with-error response — invalid HTML (duplicate IDs) resolves unpredictably. A few lines of vanilla JS move the one form node into the `<dialog>` before it opens and back to the sidebar slot when it closes.
- **The delete confirmation dialog's copy is computed server-side per row** (from the `transaction_count` `ListCategoriesWithTransactionCounts` already fetched), not via another round-trip when "Xóa" is clicked — SPEC.md's three copy variants (no transactions / expense with transactions / income with transactions, the last being the human's own resolution beyond SPEC.md) are already known at page-render time.
- **`deleteCategoryHandler`'s income-blocked and default-category paths return plain `403`/`409` with no body**, not a rendered fragment — the UI never presents a way to trigger them under normal use (a default category never shows a delete dialog at all; an income category with transactions shows a dialog with only a dismiss button, no delete action), so these only fire on a forged request and don't need a friendly rendering.

- [ ] **Step 1: Add `GetCategoryWithTransactionCount` to `internal/database/queries/categories.sql`**

Append:

```sql
-- name: GetCategoryWithTransactionCount :one
SELECT c.*, COUNT(t.id) AS transaction_count
FROM categories c
LEFT JOIN transactions t ON t.category_id = c.id AND t.user_id = $2
WHERE c.id = $1 AND (c.user_id = $2 OR c.user_id IS NULL)
GROUP BY c.id;
```

Regenerate via `sqlc generate`, or hand-append to `internal/sqlcgen/categories.sql.go` (same rationale as Task 3 Step 8):

```go
const getCategoryWithTransactionCount = `-- name: GetCategoryWithTransactionCount :one
SELECT c.id, c.user_id, c.name, c.type, c.color, c.created_at, COUNT(t.id) AS transaction_count
FROM categories c
LEFT JOIN transactions t ON t.category_id = c.id AND t.user_id = $2
WHERE c.id = $1 AND (c.user_id = $2 OR c.user_id IS NULL)
GROUP BY c.id
`

type GetCategoryWithTransactionCountParams struct {
	ID     int64       `json:"id"`
	UserID pgtype.Int8 `json:"user_id"`
}

type GetCategoryWithTransactionCountRow struct {
	ID               int64              `json:"id"`
	UserID           pgtype.Int8        `json:"user_id"`
	Name             string             `json:"name"`
	Type             string             `json:"type"`
	Color            string             `json:"color"`
	CreatedAt        pgtype.Timestamptz `json:"created_at"`
	TransactionCount int64              `json:"transaction_count"`
}

func (q *Queries) GetCategoryWithTransactionCount(ctx context.Context, arg GetCategoryWithTransactionCountParams) (GetCategoryWithTransactionCountRow, error) {
	row := q.db.QueryRow(ctx, getCategoryWithTransactionCount, arg.ID, arg.UserID)
	var i GetCategoryWithTransactionCountRow
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.Name,
		&i.Type,
		&i.Color,
		&i.CreatedAt,
		&i.TransactionCount,
	)
	return i, err
}
```

- [ ] **Step 2: Update `internal/handlers/category_handlers_test.go`**

Fix `TestCreateAndListCategories`'s form: change `"color": {"#111111"}` to `"color": {"#D97757"}` (an arbitrary free color is no longer accepted now that `categories_color_check` and the handler's own `isValidSwatch` guard both enforce the fixed palette).

Replace `TestDeleteCategoryInUseShowsFriendlyError` (its premise — blocking any in-use delete — no longer matches the new reassign-for-expense behavior) with:

```go
func TestDeleteExpenseCategoryReassignsTransactions(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-reassign@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "cat-reassign@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	category, err := deps.Queries.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: user.ID, Valid: true}, Name: "Sẽ bị gộp", Type: "expense", Color: "#D97757",
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 5000, Type: "expense",
		Description: "reassign me", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d", category.ID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tok)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting an expense category with transactions, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	moved, err := deps.Queries.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txn.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("get reassigned transaction: %v", err)
	}
	khac, err := deps.Queries.GetDefaultCategoryForReassignment(ctx)
	if err != nil {
		t.Fatalf("get Khác default: %v", err)
	}
	if moved.CategoryID != khac.ID {
		t.Fatalf("expected transaction to be reassigned to Khác (id %d), got category_id %d", khac.ID, moved.CategoryID)
	}
	if _, err := deps.Queries.GetCategoryForUser(ctx, sqlcgen.GetCategoryForUserParams{ID: category.ID, UserID: pgtype.Int8{Int64: user.ID, Valid: true}}); err == nil {
		t.Fatal("expected the deleted category to no longer exist")
	}
}

func TestDeleteIncomeCategoryWithTransactionsIsBlocked(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-income-block@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "cat-income-block@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	category, err := deps.Queries.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: user.ID, Valid: true}, Name: "Thu riêng", Type: "income", Color: "#4FA871",
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteCategory(ctx, sqlcgen.DeleteCategoryParams{ID: category.ID, UserID: pgtype.Int8{Int64: user.ID, Valid: true}})
	})

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 5000, Type: "income",
		Description: "blocks delete", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d", category.ID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tok)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 blocking delete of an income category with transactions, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if _, err := deps.Queries.GetCategoryForUser(ctx, sqlcgen.GetCategoryForUserParams{ID: category.ID, UserID: pgtype.Int8{Int64: user.ID, Valid: true}}); err != nil {
		t.Fatal("expected the blocked category to still exist")
	}
}

func TestDeleteDefaultCategoryIsForbidden(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-default-delete@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}
	var defaultCategory sqlcgen.Category
	for _, c := range categories {
		if !c.UserID.Valid {
			defaultCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/categories/%d", defaultCategory.ID), nil)
	deleteReq.AddCookie(cookie)
	withCSRF(deleteReq, tok)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 deleting a default category, got %d", deleteRec.Code)
	}
}

func TestUpdateCategoryColorAppliesToDefaultCategory(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-color@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}
	target := categories[0]
	original := target.Color
	newColor := "#8B7BD8"
	if original == newColor {
		newColor = "#5B8DEF"
	}
	t.Cleanup(func() {
		deps.Queries.UpdateCategoryColor(context.Background(), sqlcgen.UpdateCategoryColorParams{ID: target.ID, UserID: pgtype.Int8{}, Color: original})
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{"color": {newColor}}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/categories/%d/color", target.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 changing a default category's color, got %d: %s", rec.Code, rec.Body.String())
	}
	updated, err := deps.Queries.GetCategoryForUser(context.Background(), sqlcgen.GetCategoryForUserParams{ID: target.ID, UserID: pgtype.Int8{}})
	if err != nil {
		t.Fatalf("get updated category: %v", err)
	}
	if updated.Color != newColor {
		t.Fatalf("expected color %q, got %q", newColor, updated.Color)
	}
}

func TestRenameCategoryRejectsDefault(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "cat-rename-default@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("expected default categories: %v", err)
	}
	target := categories[0]

	tok := csrfTokenFor(t, router)
	form := url.Values{"name": {"Hacked Name"}}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/categories/%d/name", target.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 renaming a default category, got %d", rec.Code)
	}
}
```

- [ ] **Step 3: Run the changed/new tests to verify they fail**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestCreateAndListCategories -run TestDelete -run TestUpdateCategoryColor -run TestRenameCategory`
Expected: FAIL (route/handler code doesn't exist yet — `DELETE /categories/{id}`, `PATCH /categories/{id}/color`, `PATCH /categories/{id}/name` are all 404/405 today).

- [ ] **Step 4: Add `swatches` to `TemplateFuncs` in `internal/handlers/format.go`**

Add one entry to the returned `template.FuncMap`:

```go
"swatches": func() []string { return categorySwatches },
```

(`categorySwatches` is defined in `category_handlers.go`, Step 6 below — both files are package `handlers`, so no import is needed.)

- [ ] **Step 5: Write `internal/web/templates/category_row.html`**

```html
{{define "category_row"}}
<div id="category-row-{{.ID}}" {{if .OOBTarget}}hx-swap-oob="afterbegin:{{.OOBTarget}}"{{end}} class="flex items-center gap-3 px-[14px] py-3 border-b border-border-list last:border-b-0">
  <span class="w-[10px] h-[10px] rounded-full shrink-0" style="background-color: {{.Color}}"></span>
  <span class="text-[13px] font-medium text-ink">{{.Name}}</span>
  {{if .UserID.Valid}}
  <span class="text-[11px] text-[#A8A49D] border border-border-nav rounded-[5px] px-[6px] py-[1px]">Tự tạo</span>
  {{else}}
  <span class="text-[11px] text-[#A8A49D] border border-border-nav rounded-[5px] px-[6px] py-[1px]">Mặc định</span>
  {{end}}
  <span class="ml-auto text-[12px] text-ink-faintest font-mono">{{.TransactionCount}} giao dịch</span>
  <div class="flex items-center gap-2 relative">
    <button type="button" class="text-[12px] text-ink-faint" onclick="this.nextElementSibling.classList.toggle('hidden')">Đổi màu</button>
    <div class="hidden absolute right-0 top-6 z-10 bg-white border border-border-card rounded-lg p-2 flex flex-wrap gap-2 w-[124px] shadow-sm">
      {{range $swatch := swatches}}
      <button type="button"
        hx-patch="/categories/{{$.ID}}/color" hx-vals='{"color":"{{$swatch}}"}' hx-target="#category-row-{{$.ID}}" hx-swap="outerHTML"
        class="w-[26px] h-[26px] rounded-full flex items-center justify-center border-2 {{if eq $swatch $.Color}}border-accent{{else}}border-transparent{{end}}">
        <span class="w-[14px] h-[14px] rounded-full" style="background-color: {{$swatch}}"></span>
      </button>
      {{end}}
    </div>
    {{if .UserID.Valid}}
    <button type="button" hx-get="/categories/{{.ID}}/edit" hx-target="#category-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint">Sửa</button>
    <button type="button" onclick="document.getElementById('delete-dialog-{{.ID}}').showModal()" class="text-[12px] text-ink-faint">Xóa</button>
    <dialog id="delete-dialog-{{.ID}}" class="rounded-xl p-5 max-w-[320px] w-[85vw] backdrop:bg-black/30">
      <p class="text-[15px] font-semibold text-ink mb-2">Xóa danh mục "{{.Name}}"?</p>
      {{if and (gt .TransactionCount 0) (eq .Type "income")}}
      <p class="text-[13px] text-ink-faint mb-4">Danh mục này đang có {{.TransactionCount}} giao dịch thu. Hãy chuyển các giao dịch sang danh mục khác trước khi xóa.</p>
      <div class="flex justify-end">
        <button type="button" onclick="document.getElementById('delete-dialog-{{.ID}}').close()" class="px-4 h-9 rounded-lg border border-border-input text-[13px] text-ink">Đã hiểu</button>
      </div>
      {{else}}
      <p class="text-[13px] text-ink-faint mb-4">
        {{if gt .TransactionCount 0}}Danh mục này đang có {{.TransactionCount}} giao dịch. Các giao dịch sẽ được chuyển sang "Khác".{{else}}Hành động này không thể hoàn tác.{{end}}
      </p>
      <div class="flex gap-2 justify-end">
        <button type="button" onclick="document.getElementById('delete-dialog-{{.ID}}').close()" class="px-4 h-9 rounded-lg border border-border-input text-[13px] text-ink">Hủy</button>
        <button type="button" hx-delete="/categories/{{.ID}}" hx-target="#category-row-{{.ID}}" hx-swap="outerHTML" class="px-4 h-9 rounded-lg bg-expense text-white text-[13px] font-semibold">Xóa</button>
      </div>
      {{end}}
    </dialog>
    {{end}}
  </div>
</div>
{{end}}

{{define "category_row_edit"}}
<form id="category-row-{{.ID}}" hx-patch="/categories/{{.ID}}/name" hx-target="#category-row-{{.ID}}" hx-swap="outerHTML" class="flex items-center gap-3 px-[14px] py-3 border-b border-border-list">
  <span class="w-[10px] h-[10px] rounded-full shrink-0" style="background-color: {{.Color}}"></span>
  <input name="name" value="{{.Name}}" required class="flex-1 h-8 px-2 rounded border border-border-input text-[13px]">
  {{if .Error}}<span class="text-[12px] text-expense">{{.Error}}</span>{{end}}
  <button type="submit" class="text-[12px] text-accent font-semibold">Lưu</button>
  <button type="button" hx-get="/categories/{{.ID}}/view" hx-target="#category-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint">Hủy</button>
</form>
{{end}}
```

- [ ] **Step 6: Rewrite `internal/handlers/category_handlers.go`**

```go
package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// categorySwatches is the fixed 8-color palette SPEC.md section 1 offers in
// the category color picker. #A1A1AA (the 9th seeded color) is deliberately
// excluded here -- it's reserved for the "Khác" default and the chart's
// synthetic "Khác" aggregation, never user-selectable.
var categorySwatches = []string{
	"#D97757", "#5B8DEF", "#8B7BD8", "#6BA292",
	"#E0A82E", "#D97AA0", "#4FA871", "#7CA65C",
}

func isValidSwatch(color string) bool {
	for _, c := range categorySwatches {
		if c == color {
			return true
		}
	}
	return false
}

// categoryRowData builds the flat map category_row.html and
// category_row_edit.html both expect. Every response path that renders a
// row goes through this so the template never has to distinguish a raw
// sqlc struct from a hand-built one -- see Task 4's design notes for why
// that distinction matters for the optional OOBTarget field.
func categoryRowData(id int64, userID pgtype.Int8, name, typ, color string, txnCount int64, oobTarget string) map[string]any {
	return map[string]any{
		"ID": id, "UserID": userID, "Name": name, "Type": typ, "Color": color,
		"TransactionCount": txnCount, "OOBTarget": oobTarget,
	}
}

func categoriesPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			handleCreateCategory(w, r, deps, userID)
			return
		}

		rows, err := deps.Queries.ListCategoriesWithTransactionCounts(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}

		var expense, income []map[string]any
		hasCustom := false
		for _, row := range rows {
			data := categoryRowData(row.ID, row.UserID, row.Name, row.Type, row.Color, row.TransactionCount, "")
			if row.Type == "expense" {
				expense = append(expense, data)
			} else {
				income = append(income, data)
			}
			if row.UserID.Valid {
				hasCustom = true
			}
		}

		render(w, r, deps, "categories", "categories", map[string]any{
			"ExpenseCategories":   expense,
			"IncomeCategories":    income,
			"HasCustomCategories": hasCustom,
		})
	}
}

func handleCreateCategory(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	name := strings.TrimSpace(r.FormValue("name"))
	typ := r.FormValue("type")
	color := r.FormValue("color")

	fail := func(msg string) {
		w.Header().Set("HX-Retarget", "#add-category-form")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderNamed(w, r, deps, "categories", "add_category_form", "", map[string]any{
			"CategoryError": msg, "CategoryName": name, "CategoryType": typ,
		})
	}

	if name == "" {
		fail("Vui lòng nhập tên danh mục.")
		return
	}
	if typ != "expense" && typ != "income" {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	if !isValidSwatch(color) {
		http.Error(w, "invalid color", http.StatusBadRequest)
		return
	}

	created, err := deps.Queries.CreateCategory(r.Context(), sqlcgen.CreateCategoryParams{
		UserID: pgInt64(userID),
		Name:   name,
		Type:   typ,
		Color:  color,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail("Đã có danh mục tên này.")
			return
		}
		log.Printf("create category: %v", err)
		fail("Không thể tạo danh mục, vui lòng thử lại.")
		return
	}

	renderNamed(w, r, deps, "categories", "category_row", "", categoryRowData(
		created.ID, created.UserID, created.Name, created.Type, created.Color, 0, "#category-list-"+created.Type,
	))
}

func updateCategoryColorHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		color := r.FormValue("color")
		if !isValidSwatch(color) {
			http.Error(w, "invalid color", http.StatusBadRequest)
			return
		}

		// UpdateCategoryColor's WHERE clause matches a row owned by this
		// user OR a shared default (user_id IS NULL) -- SPEC.md section 4.1
		// explicitly lets defaults have a working "Đổi màu" action with no
		// ownership carve-out, unlike rename/delete.
		updated, err := deps.Queries.UpdateCategoryColor(r.Context(), sqlcgen.UpdateCategoryColorParams{
			ID: id, UserID: pgInt64(userID), Color: color,
		})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		count, _ := deps.Queries.CountTransactionsForCategory(r.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: updated.ID, UserID: userID})
		renderNamed(w, r, deps, "categories", "category_row", "", categoryRowData(updated.ID, updated.UserID, updated.Name, updated.Type, updated.Color, count, ""))
	}
}

func editCategoryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := deps.Queries.GetCategoryWithTransactionCount(r.Context(), sqlcgen.GetCategoryWithTransactionCountParams{ID: id, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		if !row.UserID.Valid {
			http.Error(w, "default categories cannot be renamed", http.StatusForbidden)
			return
		}
		renderNamed(w, r, deps, "categories", "category_row_edit", "", categoryRowData(row.ID, row.UserID, row.Name, row.Type, row.Color, row.TransactionCount, ""))
	}
}

func viewCategoryRowHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		row, err := deps.Queries.GetCategoryWithTransactionCount(r.Context(), sqlcgen.GetCategoryWithTransactionCountParams{ID: id, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		renderNamed(w, r, deps, "categories", "category_row", "", categoryRowData(row.ID, row.UserID, row.Name, row.Type, row.Color, row.TransactionCount, ""))
	}
}

func updateCategoryNameHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		existing, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		if !existing.UserID.Valid {
			http.Error(w, "default categories cannot be renamed", http.StatusForbidden)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			data := categoryRowData(existing.ID, existing.UserID, existing.Name, existing.Type, existing.Color, 0, "")
			data["Error"] = "Vui lòng nhập tên danh mục."
			renderNamed(w, r, deps, "categories", "category_row_edit", "", data)
			return
		}

		updated, err := deps.Queries.UpdateCategoryName(r.Context(), sqlcgen.UpdateCategoryNameParams{ID: id, UserID: pgInt64(userID), Name: name})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				data := categoryRowData(existing.ID, existing.UserID, name, existing.Type, existing.Color, 0, "")
				data["Error"] = "Đã có danh mục tên này."
				renderNamed(w, r, deps, "categories", "category_row_edit", "", data)
				return
			}
			log.Printf("update category name: %v", err)
			http.Error(w, "could not rename category", http.StatusInternalServerError)
			return
		}

		count, _ := deps.Queries.CountTransactionsForCategory(r.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: updated.ID, UserID: userID})
		renderNamed(w, r, deps, "categories", "category_row", "", categoryRowData(updated.ID, updated.UserID, updated.Name, updated.Type, updated.Color, count, ""))
	}
}

func deleteCategoryHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: id, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}
		if !category.UserID.Valid {
			http.Error(w, "default categories cannot be deleted", http.StatusForbidden)
			return
		}

		count, err := deps.Queries.CountTransactionsForCategory(r.Context(), sqlcgen.CountTransactionsForCategoryParams{CategoryID: id, UserID: userID})
		if err != nil {
			http.Error(w, "could not check category usage", http.StatusInternalServerError)
			return
		}

		if count > 0 && category.Type == "income" {
			// No income-side "Khác" exists in the 9-category default set to
			// reassign into (confirmed with the human via AskUserQuestion),
			// so an income category with existing transactions can't be
			// deleted at all.
			http.Error(w, "category has existing transactions", http.StatusConflict)
			return
		}

		if count > 0 {
			tx, err := deps.DB.Begin(r.Context())
			if err != nil {
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			defer tx.Rollback(r.Context())
			qtx := deps.Queries.WithTx(tx)

			khac, err := qtx.GetDefaultCategoryForReassignment(r.Context())
			if err != nil {
				log.Printf("delete category: load Khác default: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if _, err := qtx.ReassignCategoryTransactions(r.Context(), sqlcgen.ReassignCategoryTransactionsParams{
				CategoryID: khac.ID, CategoryID_2: id, UserID: userID,
			}); err != nil {
				log.Printf("delete category: reassign transactions: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if _, err := qtx.DeleteCategory(r.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgInt64(userID)}); err != nil {
				log.Printf("delete category: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			if err := tx.Commit(r.Context()); err != nil {
				log.Printf("delete category: commit: %v", err)
				http.Error(w, "could not delete category", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		if _, err := deps.Queries.DeleteCategory(r.Context(), sqlcgen.DeleteCategoryParams{ID: id, UserID: pgInt64(userID)}); err != nil {
			log.Printf("delete category: %v", err)
			http.Error(w, "could not delete category", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
```

- [ ] **Step 7: Update routes in `internal/handlers/router.go`**

Inside the `RequireAuth` group, replace:

```go
pr.Post("/categories/{id}/delete", deleteCategoryHandler(deps))
```

with:

```go
pr.Patch("/categories/{id}/color", updateCategoryColorHandler(deps))
pr.Get("/categories/{id}/edit", editCategoryHandler(deps))
pr.Get("/categories/{id}/view", viewCategoryRowHandler(deps))
pr.Patch("/categories/{id}/name", updateCategoryNameHandler(deps))
pr.Delete("/categories/{id}", deleteCategoryHandler(deps))
```

- [ ] **Step 8: Write `internal/web/templates/categories.html`**

```html
{{define "content"}}
<div class="max-w-[880px] mx-auto px-6 pt-7 pb-9 flex flex-col md:flex-row gap-5">
  <div class="flex-1 flex flex-col gap-5">
    <h1 class="hidden md:block text-[18px] font-semibold text-ink">Danh mục</h1>
    <div class="md:hidden flex items-center justify-between">
      <h1 class="text-[19px] font-semibold text-ink">Danh mục</h1>
      <button type="button" onclick="openAddCategorySheet()" class="w-[34px] h-[34px] rounded-full bg-accent text-white text-[18px] leading-none">＋</button>
    </div>

    <div>
      <p class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint mb-2">Chi</p>
      <div id="category-list-expense" class="bg-surface border border-border-card rounded-xl overflow-hidden">
        {{range .ExpenseCategories}}{{template "category_row" .}}{{end}}
      </div>
    </div>
    <div>
      <p class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint mb-2">Thu</p>
      <div id="category-list-income" class="bg-surface border border-border-card rounded-xl overflow-hidden">
        {{range .IncomeCategories}}{{template "category_row" .}}{{end}}
      </div>
    </div>

    {{if not .HasCustomCategories}}
    <div class="text-center py-11 px-7">
      <p class="text-[13px] text-ink-faint">Bạn chưa tạo danh mục riêng</p>
      <p class="text-[13px] text-ink-faint mt-1">9 danh mục mặc định đã sẵn sàng. Tạo thêm khi bạn cần theo dõi khoản chi riêng.</p>
      <button type="button" onclick="openAddCategorySheet()" class="md:hidden mt-3 px-4 h-9 rounded-lg border border-border-input text-[13px] text-ink">Thêm danh mục</button>
    </div>
    {{end}}
  </div>

  <div class="md:w-[300px] md:sticky md:top-[78px] md:self-start">
    <div id="add-category-slot" class="hidden md:block">
      {{template "add_category_form" .}}
    </div>
  </div>
</div>

<dialog id="add-category-sheet" class="md:hidden w-full max-w-full m-0 mt-auto mb-0 rounded-t-[20px] rounded-b-none p-0 backdrop:bg-black/32">
  <div class="p-5 pb-8" id="add-category-sheet-body"></div>
</dialog>
<script>
  function openAddCategorySheet() {
    var slot = document.getElementById('add-category-slot');
    var body = document.getElementById('add-category-sheet-body');
    if (slot.firstElementChild) body.appendChild(slot.firstElementChild);
    document.getElementById('add-category-sheet').showModal();
  }
  document.getElementById('add-category-sheet').addEventListener('close', function () {
    var slot = document.getElementById('add-category-slot');
    var body = document.getElementById('add-category-sheet-body');
    if (body.firstElementChild) slot.appendChild(body.firstElementChild);
  });
</script>
{{end}}

{{define "add_category_form"}}
<div id="add-category-form" class="bg-surface border border-border-card rounded-xl p-[18px] flex flex-col gap-[15px]">
  <p class="text-[14px] font-semibold text-ink">Thêm danh mục</p>
  <form hx-post="/categories" hx-swap="none" hx-on::after-request="if(event.detail.successful) this.querySelector('[name=name]').value=''" class="flex flex-col gap-[15px]">
    <div class="flex flex-col gap-[6px]">
      <label class="text-[12px] font-medium text-ink-muted" for="new-category-name">Tên danh mục</label>
      <input id="new-category-name" name="name" value="{{.CategoryName}}" placeholder="VD: Học phí" required class="h-9 px-3 rounded-lg border border-border-input text-[13px]">
      {{if .CategoryError}}<span class="text-[12px] text-expense">{{.CategoryError}}</span>{{end}}
    </div>
    <div class="flex flex-col gap-[6px]">
      <label class="text-[12px] font-medium text-ink-muted">Loại</label>
      <div class="flex bg-track rounded-lg p-[3px] gap-1">
        <label class="flex-1"><input type="radio" name="type" value="expense" {{if ne .CategoryType "income"}}checked{{end}} class="peer hidden"><span class="flex h-8 items-center justify-center rounded-[6px] text-[13px] peer-checked:bg-white peer-checked:font-semibold peer-checked:shadow-[0_1px_2px_rgba(0,0,0,0.06)] text-ink-muted">Chi</span></label>
        <label class="flex-1"><input type="radio" name="type" value="income" {{if eq .CategoryType "income"}}checked{{end}} class="peer hidden"><span class="flex h-8 items-center justify-center rounded-[6px] text-[13px] peer-checked:bg-white peer-checked:font-semibold peer-checked:shadow-[0_1px_2px_rgba(0,0,0,0.06)] text-ink-muted">Thu</span></label>
      </div>
    </div>
    <div class="flex flex-col gap-[6px]">
      <label class="text-[12px] font-medium text-ink-muted">Màu</label>
      <div class="flex flex-wrap gap-2">
        {{range $i, $swatch := swatches}}
        <label class="w-[26px] h-[26px] rounded-full flex items-center justify-center border-2 border-transparent has-[:checked]:border-accent">
          <input type="radio" name="color" value="{{$swatch}}" {{if eq $i 0}}checked{{end}} class="hidden">
          <span class="w-[14px] h-[14px] rounded-full" style="background-color: {{$swatch}}"></span>
        </label>
        {{end}}
      </div>
    </div>
    <button type="submit" class="h-[38px] rounded-lg bg-accent text-white text-[13px] font-semibold">Thêm danh mục</button>
    <p class="text-[12px] text-placeholder">Danh mục mặc định không thể xóa, chỉ đổi được màu.</p>
  </form>
</div>
{{end}}
```

- [ ] **Step 9: Run the full test suite to verify it passes**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

- [ ] **Step 10: Manual smoke check**

Run: `go run ./cmd/server`, visit `/categories`: create a custom category, change a default category's color, rename a custom category, delete a custom expense category that has a transaction (confirm it disappears and the transaction now shows "Khác" on `/transactions`), attempt to delete a custom income category with a transaction (confirm the dialog only offers "Đã hiểu").

---

### Task 5: Giao dịch — close the transaction validation gaps

**Files:**
- Modify: `internal/handlers/transaction_handlers.go` (`transactionsPage`'s POST branch only — the response format/markup this renders is still the old plain page at the end of this task; Task 6 rewrites both the markup and the response shape, carrying this validation logic forward unchanged)
- Modify: `internal/handlers/transaction_handlers_test.go`

Closes three gaps found while reading the existing handler: `type` isn't checked against the selected category's `type`, `description` has no length limit, and `occurred_on` accepts any date including years in the future. SPEC.md section 8 requires the first two and says dates must not be "far in the future" without a number — this plan uses **7 days** as a documented, easily-adjustable default (see Global Constraints).

- [ ] **Step 1: Add failing tests to `internal/handlers/transaction_handlers_test.go`**

```go
func TestCreateTransactionRejectsTypeCategoryMismatch(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-type-mismatch@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"income"}, // deliberately mismatched
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering with a validation error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "không khớp") {
		t.Fatalf("expected a type/category mismatch error, got: %s", rec.Body.String())
	}
}

func TestCreateTransactionRejectsLongDescription(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-long-desc@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"expense"},
		"description": {strings.Repeat("a", 201)},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering with a validation error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "200 ký tự") {
		t.Fatalf("expected a description-length error, got: %s", rec.Body.String())
	}
}

func TestCreateTransactionRejectsFarFutureDate(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-far-future@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"expense"},
		"occurred_on": {time.Now().AddDate(0, 0, 30).Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 re-rendering with a validation error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "tương lai") {
		t.Fatalf("expected a far-future-date error, got: %s", rec.Body.String())
	}
}

func TestCreateTransactionAcceptsNearFutureDate(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-near-future@example.com", "s3cret-pass")
	ctx := context.Background()

	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	var expenseCategory sqlcgen.Category
	for _, c := range categories {
		if c.Type == "expense" {
			expenseCategory = c
			break
		}
	}

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-near-future@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(expenseCategory.ID, 10)},
		"amount":      {"10000"},
		"type":        {"expense"},
		"occurred_on": {time.Now().AddDate(0, 0, 3).Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "tương lai") {
		t.Fatalf("expected a 3-day-future date (within the 7-day threshold) to be accepted, got error in body: %s", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestCreateTransactionRejects -run TestCreateTransactionAccepts`
Expected: FAIL — none of the three validations exist yet, so all four requests either succeed unexpectedly or (mismatch case) create an inconsistent row.

- [ ] **Step 3: Update `transactionsPage`'s POST branch in `internal/handlers/transaction_handlers.go`**

Replace the block from the existing category-ownership check through the `CreateTransaction` call with:

```go
			category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{
				ID:     categoryID,
				UserID: pgInt64(userID),
			})
			if err != nil {
				http.Error(w, "category not found", http.StatusForbidden)
				return
			}

			switch {
			case category.Type != txnType:
				formErr = "Loại giao dịch không khớp với danh mục đã chọn"
			case len([]rune(r.FormValue("description"))) > 200:
				formErr = "Ghi chú tối đa 200 ký tự"
			case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
				formErr = "Ngày giao dịch không được ở quá xa trong tương lai"
			default:
				_, err = deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
					UserID:      userID,
					CategoryID:  categoryID,
					Amount:      amount,
					Type:        txnType,
					Description: r.FormValue("description"),
					OccurredOn:  pgDate(occurredOn),
				})
				if err != nil {
					formErr = "could not create transaction"
				}
			}
```

(The three `formErr` assignments replace the old always-attempt-create flow; the `default` case is the only path that still calls `CreateTransaction`. `vietnamLocation` is the same package-level value `currentMonthRange` already uses, from `pg.go`.)

- [ ] **Step 4: Run the tests to verify they pass**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestCreateTransaction`
Expected: PASS.

- [ ] **Step 5: Run the full test suite**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

---

### Task 6: Giao dịch — quick-add form, list, OOB totals, category-by-type select

**Files:**
- Rewrite: `internal/handlers/transaction_handlers.go`
- Rewrite: `internal/web/templates/transactions.html`
- Create: `internal/web/templates/transaction_row.html`
- Modify: `internal/handlers/router.go` (add `GET /transactions/category-options`)
- Modify: `internal/handlers/transaction_handlers_test.go`, `internal/handlers/report_handlers_test.go`, `internal/handlers/smoke_test.go`

**Interfaces:**
- `categoryOptionsHandler(deps Deps) http.HandlerFunc` — `GET /transactions/category-options?type=expense|income`, returns the `category_options` `<select>` fragment.
- `handleCreateTransaction`, `renderTransactionsPage` — internal helpers `transactionsPage` now delegates to.

Design notes:
- **"Sửa"/"Xóa" buttons in `transaction_row.html` point at routes Task 8 creates** (`GET /transactions/{id}/edit`, the delete-confirm flow). They render inert until then — this task's own tests never click them, only Task 8's do.
- **Validation errors from the quick-add form use the same `HX-Retarget`/`HX-Reswap` technique** as the category add-form (Task 4): the form's own `hx-target` is the transaction list (for the success path's prepend), so a validation failure retargets the response to `#quick-add-form-wrapper` instead. Per-field inline errors (SPEC.md's stated ideal) are scoped out in favor of one generic error line below the form — the three validations from Task 5 are rare in normal use (wrong category type, 200+ char note, date >7 days out), and this keeps the fragment-swap wiring from ballooning; revisit if Task 12's visual QA finds it reads badly.
- **A successful create returns the new row plus three out-of-band fragments** (`#totals-summary`, `#transaction-count`, `#remaining-row`), bundled through one `transaction_create_response` template so it's still a single `render`/`renderNamed` call, not manual string concatenation.
- **`Remaining` (Còn lại tháng này) is `TotalIncome - TotalExpense`** for the currently-filtered month.
- **Task 5 introduced a type-must-match-category check that makes `categories[0]` an unsafe assumption** in tests that hardcode `"type": "expense"` — under either a byte-order or Vietnamese-aware collation, `categories[0]` from `ListCategoriesForUser`'s `ORDER BY user_id NULLS FIRST, name` happens to land on an expense category among the 9 new defaults, but relying on that is fragile. This task adds a `firstCategoryOfType` test helper and uses it everywhere a test needs "some expense category" instead of indexing blindly.

- [ ] **Step 1: Add the `firstCategoryOfType` helper and fix category selection in existing tests**

Add to `internal/handlers/transaction_handlers_test.go` (visible package-wide to every `_test.go` file in `handlers_test`):

```go
func firstCategoryOfType(t *testing.T, categories []sqlcgen.Category, typ string) sqlcgen.Category {
	t.Helper()
	for _, c := range categories {
		if c.Type == typ {
			return c
		}
	}
	t.Fatalf("expected a category of type %q", typ)
	return sqlcgen.Category{}
}
```

In `TestTransactionCRUDAndIsolation`, replace `categoryID := categories[0].ID` with `categoryID := firstCategoryOfType(t, categories, "expense").ID`.

In `internal/handlers/report_handlers_test.go`'s `TestDashboardShowsMonthlyTotal` and `internal/handlers/smoke_test.go`'s `TestEndToEndRegisterAddTransactionSeeDashboard`, replace `categories[0].ID` with `firstCategoryOfType(t, categories, "expense").ID`.

Also fix `TestTransactionCRUDAndIsolation`'s row-id scraping — the old markup had a `<form action="/transactions/{id}/delete">` to scrape an id out of; the new row markup (Step 6 below) instead carries `id="transaction-row-{id}"` on its own root element. Replace the block that builds `txnID` with:

```go
	bodyA := listRecA.Body.String()
	idx := strings.Index(bodyA, `id="transaction-row-`)
	if idx == -1 {
		t.Fatal("expected a transaction row in user A's page")
	}
	rest := bodyA[idx+len(`id="transaction-row-`):]
	endIdx := strings.Index(rest, `"`)
	if endIdx == -1 {
		t.Fatal("expected a closing quote after the transaction row id")
	}
	txnID := rest[:endIdx]
```

(The delete request built right after this still POSTs to `/transactions/{id}/delete` — that route is untouched until Task 8.)

- [ ] **Step 2: Run the affected tests to verify they still pass against the current (pre-rewrite) handler**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestTransactionCRUD -run TestDashboardShowsMonthlyTotal -run TestEndToEnd`
Expected: PASS (this step is a pure refactor of test setup, not new behavior — confirms the helper and the id-scraping fix are correct before the handler/template rewrite lands).

- [ ] **Step 3: Write `internal/web/templates/transaction_row.html`**

```html
{{define "transaction_row"}}
<div id="transaction-row-{{.ID}}" class="group flex items-center gap-4 px-4 py-[13px] border-b border-border-list last:border-b-0 hover:bg-surface-alt">
  <span class="w-[46px] shrink-0 font-mono text-[12px] text-ink-faintest">{{dateShort .OccurredOn}}</span>
  <span class="w-[150px] shrink-0 flex items-center gap-2 text-[13px] font-medium text-ink">
    <span class="w-2 h-2 rounded-full shrink-0" style="background-color: {{.CategoryColor}}"></span>
    <span class="truncate">{{.CategoryName}}</span>
  </span>
  <span class="flex-1 min-w-0 text-[13px] text-ink-faint truncate">{{.Description}}</span>
  <span class="shrink-0 flex items-center gap-1 opacity-0 group-hover:opacity-100">
    <button type="button" hx-get="/transactions/{{.ID}}/edit" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint border border-[#EAE8E4] rounded-md px-[7px] py-[3px]">Sửa</button>
    <button type="button" hx-get="/transactions/{{.ID}}/delete-confirm" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint border border-[#EAE8E4] rounded-md px-[7px] py-[3px]">Xóa</button>
  </span>
  <span class="w-[132px] shrink-0 text-right font-mono text-[14px] font-medium {{if eq .Type "expense"}}text-expense{{else}}text-income{{end}}">{{vndSigned .Amount .Type}}</span>
</div>
{{end}}

{{define "totals_oob"}}
<div id="totals-summary" hx-swap-oob="true" class="flex items-center gap-[18px]">
  <span class="text-[13px] text-ink-faint">Chi <span class="font-mono font-medium text-expense">{{vnd .TotalExpense}}</span></span>
  <span class="text-[13px] text-ink-faint">Thu <span class="font-mono font-medium text-income">{{vnd .TotalIncome}}</span></span>
</div>
<span id="transaction-count" hx-swap-oob="true" class="text-[12px] text-ink-faint">{{.Count}} giao dịch</span>
<div id="remaining-row" hx-swap-oob="true" class="flex items-center justify-between px-4 py-[13px] bg-surface-alt">
  <span class="text-[13px] font-semibold text-ink">Còn lại tháng này</span>
  <span class="w-[132px] text-right font-mono text-[15px] font-semibold {{if lt .Remaining 0}}text-expense{{else}}text-ink{{end}}">{{vndBalance .Remaining}}</span>
</div>
{{end}}

{{define "transaction_create_response"}}{{template "transaction_row" .Row}}{{template "totals_oob" .Totals}}{{end}}
```

- [ ] **Step 4: Rewrite `internal/web/templates/transactions.html`**

```html
{{define "content"}}
<div class="max-w-[880px] mx-auto px-6 pt-7 pb-9 flex flex-col gap-[18px]">

  <div id="quick-add-form-wrapper">
    {{template "quick_add_form" .}}
  </div>

  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <button type="button" class="h-8 px-3 rounded-lg border border-border-input bg-white text-[13px] font-medium text-ink flex items-center gap-1">{{.MonthLabel}} <span class="text-placeholder">▾</span></button>
      <span id="transaction-count" class="text-[12px] text-ink-faint">{{len .Transactions}} giao dịch</span>
    </div>
    <div id="totals-summary" class="flex items-center gap-[18px]">
      <span class="text-[13px] text-ink-faint">Chi <span class="font-mono font-medium text-expense">{{vnd .TotalExpense}}</span></span>
      <span class="text-[13px] text-ink-faint">Thu <span class="font-mono font-medium text-income">{{vnd .TotalIncome}}</span></span>
    </div>
  </div>

  <div class="bg-surface border border-border-card rounded-xl overflow-hidden">
    <div id="transaction-list">
      {{range .Transactions}}{{template "transaction_row" .}}{{end}}
    </div>
    {{if not .Transactions}}
    <div class="text-center py-11 px-7">
      <p class="text-[15px] font-semibold text-ink">Chưa có giao dịch nào trong {{.MonthLabelLower}}</p>
      <p class="text-[13px] text-ink-faint mt-1 max-w-[300px] mx-auto">Thêm giao dịch đầu tiên bằng form phía trên, hoặc chọn tháng khác để xem lại.</p>
    </div>
    {{end}}
    <div id="remaining-row" class="flex items-center justify-between px-4 py-[13px] bg-surface-alt">
      <span class="text-[13px] font-semibold text-ink">Còn lại tháng này</span>
      <span class="w-[132px] text-right font-mono text-[15px] font-semibold {{if lt .Remaining 0}}text-expense{{else}}text-ink{{end}}">{{vndBalance .Remaining}}</span>
    </div>
  </div>
</div>
{{end}}

{{define "quick_add_form"}}
<div class="bg-surface border border-border-card rounded-xl p-4">
  <form id="quick-add-form" hx-post="/transactions" hx-target="#transaction-list" hx-swap="afterbegin"
    hx-on::after-request="if(event.detail.successful){ this.querySelector('[name=amount]').value=''; this.querySelector('[name=description]').value=''; }"
    class="flex items-end gap-[10px] flex-wrap">
    <div class="flex flex-col gap-[6px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Loại</label>
      <div class="flex bg-track rounded-lg p-[3px] gap-1 h-9">
        <label class="px-3 flex items-center cursor-pointer">
          <input type="radio" name="type" value="expense" {{if ne .SelectedType "income"}}checked{{end}} class="peer hidden"
            hx-get="/transactions/category-options" hx-target="#category-select" hx-trigger="change" hx-vals='{"type":"expense"}'>
          <span class="text-[13px] peer-checked:font-semibold">Chi</span>
        </label>
        <label class="px-3 flex items-center cursor-pointer">
          <input type="radio" name="type" value="income" {{if eq .SelectedType "income"}}checked{{end}} class="peer hidden"
            hx-get="/transactions/category-options" hx-target="#category-select" hx-trigger="change" hx-vals='{"type":"income"}'>
          <span class="text-[13px] peer-checked:font-semibold">Thu</span>
        </label>
      </div>
    </div>
    <div class="flex flex-col gap-[6px] w-[168px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Danh mục</label>
      {{template "category_options" .}}
    </div>
    <div class="flex flex-col gap-[6px] w-[150px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Số tiền</label>
      <input name="amount" type="number" min="1" required class="h-9 px-3 rounded-lg border border-border-input font-mono text-[14px] font-medium">
    </div>
    <div class="flex flex-col gap-[6px] w-[128px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Ngày</label>
      <input name="occurred_on" type="date" value="{{.Today}}" required class="h-9 px-3 rounded-lg border border-border-input font-mono text-[13px]">
    </div>
    <div class="flex flex-col gap-[6px] flex-1 min-w-[140px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Ghi chú</label>
      <input name="description" type="text" placeholder="Không bắt buộc" class="h-9 px-3 rounded-lg border border-border-input text-[13px]">
    </div>
    <button type="submit" class="h-9 px-[18px] rounded-lg bg-accent text-white text-[13px] font-semibold">Thêm</button>
  </form>
  {{if .QuickAddError}}<p class="text-[12px] text-expense mt-2">{{.QuickAddError}}</p>{{end}}
</div>
{{end}}

{{define "category_options"}}
<select id="category-select" name="category_id" class="h-9 px-3 rounded-lg border border-border-input text-[13px] bg-white">
  {{range .Categories}}<option value="{{.ID}}" style="color: {{.Color}}">● {{.Name}}</option>{{end}}
</select>
{{end}}
```

- [ ] **Step 5: Add `GET /transactions/category-options` to `internal/handlers/router.go`**

Inside the `RequireAuth` group, add:

```go
pr.Get("/transactions/category-options", categoryOptionsHandler(deps))
```

- [ ] **Step 6: Rewrite `internal/handlers/transaction_handlers.go`**

```go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgDate converts a parsed calendar date into the pgtype.Date that sqlc
// generates for a DATE column.
func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func monthLabel(t time.Time) string      { return fmt.Sprintf("Tháng %d, %d", int(t.Month()), t.Year()) }
func monthLabelLower(t time.Time) string { return fmt.Sprintf("tháng %d", int(t.Month())) }

func transactionsPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			handleCreateTransaction(w, r, deps, userID)
			return
		}

		renderTransactionsPage(w, r, deps, userID, "", "")
	}
}

func renderTransactionsPage(w http.ResponseWriter, r *http.Request, deps Deps, userID int64, quickAddError, selectedType string) {
	from, to := currentMonthRange()

	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
	})
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return
	}

	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
	})
	if err != nil {
		http.Error(w, "could not load totals", http.StatusInternalServerError)
		return
	}

	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		http.Error(w, "could not load categories", http.StatusInternalServerError)
		return
	}
	typ := selectedType
	if typ != "income" {
		typ = "expense"
	}
	var filteredCategories []sqlcgen.Category
	for _, c := range allCategories {
		if c.Type == typ {
			filteredCategories = append(filteredCategories, c)
		}
	}

	render(w, r, deps, "transactions", "transactions", map[string]any{
		"Transactions":    transactions,
		"Categories":      filteredCategories,
		"SelectedType":    selectedType,
		"TotalExpense":    totals.TotalExpense,
		"TotalIncome":     totals.TotalIncome,
		"Remaining":       totals.TotalIncome - totals.TotalExpense,
		"MonthLabel":      monthLabel(from.Time),
		"MonthLabelLower": monthLabelLower(from.Time),
		"Today":           time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError":   quickAddError,
	})
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid category", http.StatusBadRequest)
		return
	}
	amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if err != nil || amount <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}
	occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}
	txnType := r.FormValue("type")
	if txnType != "expense" && txnType != "income" {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}

	category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{
		ID: categoryID, UserID: pgInt64(userID),
	})
	if err != nil {
		http.Error(w, "category not found", http.StatusForbidden)
		return
	}

	var formErr string
	switch {
	case category.Type != txnType:
		formErr = "Loại giao dịch không khớp với danh mục đã chọn"
	case len([]rune(r.FormValue("description"))) > 200:
		formErr = "Ghi chú tối đa 200 ký tự"
	case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
		formErr = "Ngày giao dịch không được ở quá xa trong tương lai"
	}
	if formErr != "" {
		w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderTransactionsPageForm(w, r, deps, userID, formErr, txnType)
		return
	}

	created, err := deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
		UserID: userID, CategoryID: categoryID, Amount: amount, Type: txnType,
		Description: r.FormValue("description"), OccurredOn: pgDate(occurredOn),
	})
	if err != nil {
		w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderTransactionsPageForm(w, r, deps, userID, "Không thể thêm giao dịch, vui lòng thử lại.", txnType)
		return
	}

	from, to := currentMonthRange()
	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load totals", http.StatusInternalServerError)
		return
	}
	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return
	}

	renderNamed(w, r, deps, "transactions", "transaction_create_response", "", map[string]any{
		"Row": map[string]any{
			"ID": created.ID, "CategoryName": category.Name, "CategoryColor": category.Color,
			"Description": created.Description, "OccurredOn": created.OccurredOn,
			"Amount": created.Amount, "Type": created.Type,
		},
		"Totals": map[string]any{
			"TotalExpense": totals.TotalExpense, "TotalIncome": totals.TotalIncome,
			"Remaining": totals.TotalIncome - totals.TotalExpense, "Count": len(transactions),
		},
	})
}

// renderTransactionsPageForm re-renders just the quick_add_form fragment
// (targeted via HX-Retarget by the caller) after a validation failure,
// with the category list filtered to match the type the user had selected
// so their in-progress selection stays coherent.
func renderTransactionsPageForm(w http.ResponseWriter, r *http.Request, deps Deps, userID int64, errMsg, selectedType string) {
	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		http.Error(w, "could not load categories", http.StatusInternalServerError)
		return
	}
	var filteredCategories []sqlcgen.Category
	for _, c := range allCategories {
		if c.Type == selectedType {
			filteredCategories = append(filteredCategories, c)
		}
	}
	renderNamed(w, r, deps, "transactions", "quick_add_form", "", map[string]any{
		"Categories":    filteredCategories,
		"SelectedType":  selectedType,
		"Today":         time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError": errMsg,
	})
}

func categoryOptionsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		typ := r.FormValue("type")
		if typ != "income" {
			typ = "expense"
		}
		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		var filtered []sqlcgen.Category
		for _, c := range categories {
			if c.Type == typ {
				filtered = append(filtered, c)
			}
		}
		renderNamed(w, r, deps, "transactions", "category_options", "", map[string]any{"Categories": filtered})
	}
}

func deleteTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if _, err := deps.Queries.DeleteTransaction(r.Context(), sqlcgen.DeleteTransactionParams{
			ID:     id,
			UserID: userID,
		}); err != nil {
			http.Error(w, "could not delete transaction", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}
```

(`deleteTransactionHandler` is reproduced unchanged from before — Task 8 rewrites it into the inline delete-confirm + `DELETE` route flow. `renderTransactionsPage`'s old 5-arg-less call sites in `transactionsPage` above use `renderTransactionsPage(w, r, deps, userID, "", "")` for the plain GET case.)

- [ ] **Step 7: Run the full test suite to verify it passes**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

- [ ] **Step 8: Manual smoke check**

Run: `go run ./cmd/server`, visit `/transactions`: add an expense, confirm it prepends to the list and the Chi total / count / "Còn lại" row all update without a page reload; switch Loại to Thu and confirm the danh mục select reloads to income-only categories; submit with a description over 200 characters and confirm the generic error line appears without the page reloading.

---

### Task 7: Giao dịch — month filter dropdown (`?thang=YYYY-MM`) with `hx-push-url`

**Files:**
- Rewrite: `internal/handlers/transaction_handlers.go` (supersedes Task 6's version wholesale — the month-parameterization touches most of the same functions)
- Rewrite: `internal/web/templates/transactions.html` (supersedes Task 6's version — `quick_add_form`/`category_options` are unchanged, reproduced here only because this is a full-file rewrite)
- Modify: `internal/handlers/transaction_handlers_test.go`

**Interfaces:**
- `monthRangeFor(param string) (from, to pgtype.Date)` — parses a `"2006-01"` value, falling back to `currentMonthRange()` when empty/malformed.
- `buildTransactionsPageData(r, deps, userID, monthParam, quickAddError, selectedType) (map[string]any, error)` — the single data-building path both the full-page render and the month-dropdown fragment render go through.

Design notes: the month dropdown's `hx-get` always carries `HX-Request: true` (it's htmx-triggered), so `transactionsPage`'s GET branch checks that header the same way Task 2's auth tab-switch and Task 6 already established the pattern for — fragment (`transactions_month_section`) if present, full page otherwise. `hx-push-url="true"` lets htmx push the request's own resolved URL (`/transactions?thang=2026-07`) rather than needing it spelled out a second time, so `/transactions?thang=...` stays bookmarkable/back-button-able. The dropdown's "Tháng này" entry is always pinned first; months from `ListDistinctTransactionMonths` that happen to equal the current month are filtered out of the rest of the list to avoid a duplicate-looking entry.

- [ ] **Step 1: Add failing tests to `internal/handlers/transaction_handlers_test.go`**

```go
func TestTransactionsPageFiltersByMonthParam(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-filter@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-month-filter@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	pastMonth := time.Now().AddDate(0, -2, 0)
	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 7000, Type: "expense",
		Description: "Old month txn",
		OccurredOn:  pgtype.Date{Time: time.Date(pastMonth.Year(), pastMonth.Month(), 10, 0, 0, 0, 0, time.UTC), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	currentReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	currentReq.AddCookie(cookie)
	currentRec := httptest.NewRecorder()
	router.ServeHTTP(currentRec, currentReq)
	if strings.Contains(currentRec.Body.String(), "Old month txn") {
		t.Fatal("expected the current-month view to NOT include a transaction from two months ago")
	}

	monthParam := pastMonth.Format("2006-01")
	pastReq := httptest.NewRequest(http.MethodGet, "/transactions?thang="+monthParam, nil)
	pastReq.AddCookie(cookie)
	pastRec := httptest.NewRecorder()
	router.ServeHTTP(pastRec, pastReq)
	if !strings.Contains(pastRec.Body.String(), "Old month txn") {
		t.Fatalf("expected ?thang=%s to include the past-month transaction, got: %s", monthParam, pastRec.Body.String())
	}
}

func TestMonthDropdownReturnsFragmentOnly(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-month-fragment@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions?thang="+time.Now().Format("2006-01"), nil)
	req.AddCookie(cookie)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("expected a fragment response with no <html> wrapper, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `id="transactions-month-section"`) {
		t.Fatalf("expected the transactions_month_section fragment, got: %s", rec.Body.String())
	}
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestTransactionsPageFiltersByMonthParam -run TestMonthDropdownReturnsFragmentOnly`
Expected: FAIL — `?thang=` is currently ignored (`transactionsPage` always uses `currentMonthRange()`).

- [ ] **Step 3: Rewrite `internal/handlers/transaction_handlers.go`**

```go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgDate converts a parsed calendar date into the pgtype.Date that sqlc
// generates for a DATE column.
func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func monthLabel(t time.Time) string      { return fmt.Sprintf("Tháng %d, %d", int(t.Month()), t.Year()) }
func monthLabelLower(t time.Time) string { return fmt.Sprintf("tháng %d", int(t.Month())) }

// monthRangeFor returns the [from, to) bounds for the "YYYY-MM" value the
// month dropdown sends via ?thang=, falling back to the current Vietnam-
// local month when param is empty or malformed.
func monthRangeFor(param string) (from, to pgtype.Date) {
	t, err := time.ParseInLocation("2006-01", param, vietnamLocation)
	if err != nil {
		return currentMonthRange()
	}
	fromTime := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, vietnamLocation)
	return pgDate(fromTime), pgDate(fromTime.AddDate(0, 1, 0))
}

func transactionsPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		if r.Method == http.MethodPost {
			handleCreateTransaction(w, r, deps, userID)
			return
		}

		data, err := buildTransactionsPageData(r, deps, userID, r.URL.Query().Get("thang"), "", "")
		if err != nil {
			http.Error(w, "could not load transactions", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			renderNamed(w, r, deps, "transactions", "transactions_month_section", "", data)
			return
		}
		render(w, r, deps, "transactions", "transactions", data)
	}
}

// buildTransactionsPageData loads everything transactions.html (both the
// full page and the transactions_month_section fragment the month dropdown
// swaps in) needs: the selected month's transactions/totals, the dropdown's
// list of other months with data, and the quick-add form's own state.
func buildTransactionsPageData(r *http.Request, deps Deps, userID int64, monthParam, quickAddError, selectedType string) (map[string]any, error) {
	from, to := monthRangeFor(monthParam)

	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
	})
	if err != nil {
		return nil, err
	}

	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
	})
	if err != nil {
		return nil, err
	}

	months, err := deps.Queries.ListDistinctTransactionMonths(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	currentFrom, _ := currentMonthRange()
	var available []map[string]any
	for _, m := range months {
		if m.Time.Year() == currentFrom.Time.Year() && m.Time.Month() == currentFrom.Time.Month() {
			continue // already offered as the pinned "Tháng này" entry
		}
		available = append(available, map[string]any{
			"Value": m.Time.Format("2006-01"),
			"Label": monthLabel(m.Time),
		})
	}

	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		return nil, err
	}
	formType := selectedType
	if formType != "income" {
		formType = "expense"
	}
	var formCategories []sqlcgen.Category
	for _, c := range allCategories {
		if c.Type == formType {
			formCategories = append(formCategories, c)
		}
	}

	return map[string]any{
		"Transactions":      transactions,
		"TotalExpense":      totals.TotalExpense,
		"TotalIncome":       totals.TotalIncome,
		"Remaining":         totals.TotalIncome - totals.TotalExpense,
		"MonthLabel":        monthLabel(from.Time),
		"MonthLabelLower":   monthLabelLower(from.Time),
		"CurrentMonthValue": currentFrom.Time.Format("2006-01"),
		"AvailableMonths":   available,
		"Categories":        formCategories,
		"SelectedType":      selectedType,
		"Today":             time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError":     quickAddError,
	}, nil
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid category", http.StatusBadRequest)
		return
	}
	amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if err != nil || amount <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}
	occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}
	txnType := r.FormValue("type")
	if txnType != "expense" && txnType != "income" {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}

	category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{
		ID: categoryID, UserID: pgInt64(userID),
	})
	if err != nil {
		http.Error(w, "category not found", http.StatusForbidden)
		return
	}

	var formErr string
	switch {
	case category.Type != txnType:
		formErr = "Loại giao dịch không khớp với danh mục đã chọn"
	case len([]rune(r.FormValue("description"))) > 200:
		formErr = "Ghi chú tối đa 200 ký tự"
	case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
		formErr = "Ngày giao dịch không được ở quá xa trong tương lai"
	}
	if formErr != "" {
		w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderTransactionsPageForm(w, r, deps, userID, formErr, txnType)
		return
	}

	created, err := deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
		UserID: userID, CategoryID: categoryID, Amount: amount, Type: txnType,
		Description: r.FormValue("description"), OccurredOn: pgDate(occurredOn),
	})
	if err != nil {
		w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderTransactionsPageForm(w, r, deps, userID, "Không thể thêm giao dịch, vui lòng thử lại.", txnType)
		return
	}

	from, to := currentMonthRange()
	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load totals", http.StatusInternalServerError)
		return
	}
	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return
	}

	renderNamed(w, r, deps, "transactions", "transaction_create_response", "", map[string]any{
		"Row": map[string]any{
			"ID": created.ID, "CategoryName": category.Name, "CategoryColor": category.Color,
			"Description": created.Description, "OccurredOn": created.OccurredOn,
			"Amount": created.Amount, "Type": created.Type,
		},
		"Totals": map[string]any{
			"TotalExpense": totals.TotalExpense, "TotalIncome": totals.TotalIncome,
			"Remaining": totals.TotalIncome - totals.TotalExpense, "Count": len(transactions),
		},
	})
}

// renderTransactionsPageForm re-renders just the quick_add_form fragment
// (targeted via HX-Retarget by the caller) after a validation failure, with
// the category list filtered to match the type the user had selected.
func renderTransactionsPageForm(w http.ResponseWriter, r *http.Request, deps Deps, userID int64, errMsg, selectedType string) {
	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		http.Error(w, "could not load categories", http.StatusInternalServerError)
		return
	}
	var filteredCategories []sqlcgen.Category
	for _, c := range allCategories {
		if c.Type == selectedType {
			filteredCategories = append(filteredCategories, c)
		}
	}
	renderNamed(w, r, deps, "transactions", "quick_add_form", "", map[string]any{
		"Categories":    filteredCategories,
		"SelectedType":  selectedType,
		"Today":         time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError": errMsg,
	})
}

func categoryOptionsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		typ := r.FormValue("type")
		if typ != "income" {
			typ = "expense"
		}
		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		var filtered []sqlcgen.Category
		for _, c := range categories {
			if c.Type == typ {
				filtered = append(filtered, c)
			}
		}
		renderNamed(w, r, deps, "transactions", "category_options", "", map[string]any{"Categories": filtered})
	}
}

func deleteTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if _, err := deps.Queries.DeleteTransaction(r.Context(), sqlcgen.DeleteTransactionParams{
			ID:     id,
			UserID: userID,
		}); err != nil {
			http.Error(w, "could not delete transaction", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/transactions", http.StatusSeeOther)
	}
}
```

(`deleteTransactionHandler` is still the pre-Task-8 version — Task 8 replaces it.)

- [ ] **Step 4: Rewrite `internal/web/templates/transactions.html`**

```html
{{define "content"}}
<div class="max-w-[880px] mx-auto px-6 pt-7 pb-9 flex flex-col gap-[18px]">
  <div id="quick-add-form-wrapper">{{template "quick_add_form" .}}</div>
  {{template "transactions_month_section" .}}
</div>
{{end}}

{{define "transactions_month_section"}}
<div id="transactions-month-section" class="flex flex-col gap-[18px]">
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2 relative">
      <button type="button" onclick="this.nextElementSibling.classList.toggle('hidden')" class="h-8 px-3 rounded-lg border border-border-input bg-white text-[13px] font-medium text-ink flex items-center gap-1">{{.MonthLabel}} <span class="text-placeholder">▾</span></button>
      <div class="hidden absolute left-0 top-9 z-20 bg-white border border-border-card rounded-lg py-1 w-[160px] shadow-sm">
        <button type="button" hx-get="/transactions?thang={{.CurrentMonthValue}}" hx-target="#transactions-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">Tháng này</button>
        {{range .AvailableMonths}}
        <button type="button" hx-get="/transactions?thang={{.Value}}" hx-target="#transactions-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">{{.Label}}</button>
        {{end}}
      </div>
      <span id="transaction-count" class="text-[12px] text-ink-faint">{{len .Transactions}} giao dịch</span>
    </div>
    <div id="totals-summary" class="flex items-center gap-[18px]">
      <span class="text-[13px] text-ink-faint">Chi <span class="font-mono font-medium text-expense">{{vnd .TotalExpense}}</span></span>
      <span class="text-[13px] text-ink-faint">Thu <span class="font-mono font-medium text-income">{{vnd .TotalIncome}}</span></span>
    </div>
  </div>

  <div class="bg-surface border border-border-card rounded-xl overflow-hidden">
    <div id="transaction-list">
      {{range .Transactions}}{{template "transaction_row" .}}{{end}}
    </div>
    {{if not .Transactions}}
    <div class="text-center py-11 px-7">
      <p class="text-[15px] font-semibold text-ink">Chưa có giao dịch nào trong {{.MonthLabelLower}}</p>
      <p class="text-[13px] text-ink-faint mt-1 max-w-[300px] mx-auto">Thêm giao dịch đầu tiên bằng form phía trên, hoặc chọn tháng khác để xem lại.</p>
    </div>
    {{end}}
    <div id="remaining-row" class="flex items-center justify-between px-4 py-[13px] bg-surface-alt">
      <span class="text-[13px] font-semibold text-ink">Còn lại tháng này</span>
      <span class="w-[132px] text-right font-mono text-[15px] font-semibold {{if lt .Remaining 0}}text-expense{{else}}text-ink{{end}}">{{vndBalance .Remaining}}</span>
    </div>
  </div>
</div>
{{end}}

{{define "quick_add_form"}}
<div class="bg-surface border border-border-card rounded-xl p-4">
  <form id="quick-add-form" hx-post="/transactions" hx-target="#transaction-list" hx-swap="afterbegin"
    hx-on::after-request="if(event.detail.successful){ this.querySelector('[name=amount]').value=''; this.querySelector('[name=description]').value=''; }"
    class="flex items-end gap-[10px] flex-wrap">
    <div class="flex flex-col gap-[6px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Loại</label>
      <div class="flex bg-track rounded-lg p-[3px] gap-1 h-9">
        <label class="px-3 flex items-center cursor-pointer">
          <input type="radio" name="type" value="expense" {{if ne .SelectedType "income"}}checked{{end}} class="peer hidden"
            hx-get="/transactions/category-options" hx-target="#category-select" hx-trigger="change" hx-vals='{"type":"expense"}'>
          <span class="text-[13px] peer-checked:font-semibold">Chi</span>
        </label>
        <label class="px-3 flex items-center cursor-pointer">
          <input type="radio" name="type" value="income" {{if eq .SelectedType "income"}}checked{{end}} class="peer hidden"
            hx-get="/transactions/category-options" hx-target="#category-select" hx-trigger="change" hx-vals='{"type":"income"}'>
          <span class="text-[13px] peer-checked:font-semibold">Thu</span>
        </label>
      </div>
    </div>
    <div class="flex flex-col gap-[6px] w-[168px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Danh mục</label>
      {{template "category_options" .}}
    </div>
    <div class="flex flex-col gap-[6px] w-[150px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Số tiền</label>
      <input name="amount" type="number" min="1" required class="h-9 px-3 rounded-lg border border-border-input font-mono text-[14px] font-medium">
    </div>
    <div class="flex flex-col gap-[6px] w-[128px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Ngày</label>
      <input name="occurred_on" type="date" value="{{.Today}}" required class="h-9 px-3 rounded-lg border border-border-input font-mono text-[13px]">
    </div>
    <div class="flex flex-col gap-[6px] flex-1 min-w-[140px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Ghi chú</label>
      <input name="description" type="text" placeholder="Không bắt buộc" class="h-9 px-3 rounded-lg border border-border-input text-[13px]">
    </div>
    <button type="submit" class="h-9 px-[18px] rounded-lg bg-accent text-white text-[13px] font-semibold">Thêm</button>
  </form>
  {{if .QuickAddError}}<p class="text-[12px] text-expense mt-2">{{.QuickAddError}}</p>{{end}}
</div>
{{end}}

{{define "category_options"}}
<select id="category-select" name="category_id" class="h-9 px-3 rounded-lg border border-border-input text-[13px] bg-white">
  {{range .Categories}}<option value="{{.ID}}" style="color: {{.Color}}">● {{.Name}}</option>{{end}}
</select>
{{end}}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestTransactionsPageFiltersByMonthParam -run TestMonthDropdownReturnsFragmentOnly`
Expected: PASS.

- [ ] **Step 6: Run the full test suite**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

---

### Task 8: Giao dịch — inline edit and inline delete-confirm

**Files:**
- Modify: `internal/database/queries/transactions.sql` + `internal/sqlcgen/transactions.sql.go` (one more query)
- Modify: `internal/handlers/transaction_handlers.go` (add edit/view/delete-confirm/update handlers, rewrite `deleteTransactionHandler`)
- Modify: `internal/web/templates/transaction_row.html` (add `transaction_row_edit`, `transaction_row_delete_confirm`)
- Modify: `internal/handlers/router.go`
- Modify: `internal/handlers/transaction_handlers_test.go`

**Interfaces:**
- `GET /transactions/{id}/edit` → inline edit row fragment. `GET /transactions/{id}/view` → back to the display row (cancel, from either edit or delete-confirm). `GET /transactions/{id}/delete-confirm` → inline confirm bar. `PATCH /transactions/{id}` → commits the edit. `DELETE /transactions/{id}` → commits the delete (replaces the old `POST /transactions/{id}/delete`).

Design notes:
- **Inline edit only covers `category_id`, `amount`, `occurred_on`, `description` — not `type`.** SPEC.md's row layout has no "Loại" column to edit in place, and letting type change independently of category reopens the same type/category-mismatch ambiguity Task 5 closed for create. The category `<select>` in edit mode is pre-filtered to the transaction's existing type, and a submitted `category_id` whose type doesn't match is rejected with the same "không khớp" message as create.
- **The create and update success responses share one `transaction_create_response` template block** (a row + the three OOB total fragments) — despite the name, it's generic and Task 8 reuses it as-is rather than introducing a near-duplicate block.
- **Delete's response is *only* the three OOB blocks, no primary content.** The confirm button's `hx-target`/`hx-swap="outerHTML"` points at the row itself, so an empty primary response body removes it — the same pattern the categories delete flow (Task 4) already established.

- [ ] **Step 1: Add `GetTransactionWithCategory` to `internal/database/queries/transactions.sql`**

Append:

```sql
-- name: GetTransactionWithCategory :one
SELECT t.*, c.name AS category_name, c.color AS category_color
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.id = $1 AND t.user_id = $2;
```

Regenerate via `sqlc generate`, or hand-append to `internal/sqlcgen/transactions.sql.go`:

```go
const getTransactionWithCategory = `-- name: GetTransactionWithCategory :one
SELECT t.id, t.user_id, t.category_id, t.amount, t.type, t.description, t.occurred_on, t.created_at, t.updated_at, c.name AS category_name, c.color AS category_color
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.id = $1 AND t.user_id = $2
`

type GetTransactionWithCategoryParams struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`
}

type GetTransactionWithCategoryRow struct {
	ID            int64              `json:"id"`
	UserID        int64              `json:"user_id"`
	CategoryID    int64              `json:"category_id"`
	Amount        int64              `json:"amount"`
	Type          string             `json:"type"`
	Description   string             `json:"description"`
	OccurredOn    pgtype.Date        `json:"occurred_on"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
	UpdatedAt     pgtype.Timestamptz `json:"updated_at"`
	CategoryName  string             `json:"category_name"`
	CategoryColor string             `json:"category_color"`
}

func (q *Queries) GetTransactionWithCategory(ctx context.Context, arg GetTransactionWithCategoryParams) (GetTransactionWithCategoryRow, error) {
	row := q.db.QueryRow(ctx, getTransactionWithCategory, arg.ID, arg.UserID)
	var i GetTransactionWithCategoryRow
	err := row.Scan(
		&i.ID, &i.UserID, &i.CategoryID, &i.Amount, &i.Type, &i.Description,
		&i.OccurredOn, &i.CreatedAt, &i.UpdatedAt, &i.CategoryName, &i.CategoryColor,
	)
	return i, err
}
```

- [ ] **Step 2: Add `fmt` to `internal/handlers/transaction_handlers_test.go`'s imports and fix the existing delete request**

Add `"fmt"` to the import block. In `TestTransactionCRUDAndIsolation`, replace the cross-user delete request (which used the now-removed `POST /transactions/{id}/delete` route) with the new `DELETE /transactions/{id}`:

```go
	tokDel := csrfTokenFor(t, router)
	deleteReq := httptest.NewRequest(http.MethodDelete, "/transactions/"+txnID, nil)
	deleteReq.AddCookie(cookieB)
	withCSRF(deleteReq, tokDel)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
```

- [ ] **Step 3: Add new tests to `internal/handlers/transaction_handlers_test.go`**

```go
func TestUpdateTransactionAppliesEdit(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-update@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-update@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 10000, Type: "expense",
		Description: "before edit", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() {
		deps.Queries.DeleteTransaction(ctx, sqlcgen.DeleteTransactionParams{ID: txn.ID, UserID: user.ID})
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"25000"},
		"description": {"after edit"},
		"occurred_on": {time.Now().Format("2006-01-02")},
	}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transactions/%d", txn.ID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 updating a transaction, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "after edit") {
		t.Fatalf("expected updated description in response, got: %s", rec.Body.String())
	}

	updated, err := deps.Queries.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txn.ID, UserID: user.ID})
	if err != nil {
		t.Fatalf("get updated transaction: %v", err)
	}
	if updated.Amount != 25000 {
		t.Fatalf("expected amount 25000, got %d", updated.Amount)
	}
}

func TestDeleteTransactionRemovesRow(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-delete-new@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-delete-new@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	txn, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 9000, Type: "expense",
		Description: "to delete", OccurredOn: pgtype.Date{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/transactions/%d", txn.ID), nil)
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting a transaction, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := deps.Queries.GetTransaction(ctx, sqlcgen.GetTransactionParams{ID: txn.ID, UserID: user.ID}); err == nil {
		t.Fatal("expected the transaction to no longer exist")
	}
}
```

- [ ] **Step 4: Run the tests to verify they fail**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestUpdateTransaction -run TestDeleteTransaction -run TestTransactionCRUD`
Expected: FAIL — `PATCH /transactions/{id}` doesn't exist and `POST /transactions/{id}/delete` (used by the pre-fix cross-user test) is about to be removed.

- [ ] **Step 5: Add `transaction_row_edit` and `transaction_row_delete_confirm` to `internal/web/templates/transaction_row.html`**

Append:

```html
{{define "transaction_row_edit"}}
<form id="transaction-row-{{.ID}}" hx-patch="/transactions/{{.ID}}" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="flex items-center gap-4 px-4 py-[10px] border-b border-border-list last:border-b-0 bg-surface-alt">
  <input name="occurred_on" type="date" value="{{.OccurredOnValue}}" required class="w-[46px] shrink-0 font-mono text-[11px] border border-border-input rounded px-1 h-8">
  <select name="category_id" class="w-[150px] shrink-0 text-[13px] border border-border-input rounded px-1 h-8">
    {{range .CategoryOptions}}<option value="{{.ID}}" {{if eq .ID $.CategoryID}}selected{{end}}>{{.Name}}</option>{{end}}
  </select>
  <input name="description" type="text" value="{{.Description}}" class="flex-1 min-w-0 text-[13px] border border-border-input rounded px-2 h-8">
  {{if .Error}}<span class="text-[12px] text-expense shrink-0">{{.Error}}</span>{{end}}
  <span class="shrink-0 flex items-center gap-1">
    <button type="submit" class="text-[12px] text-accent font-semibold border border-border-input rounded-md px-[7px] py-[3px]">Lưu</button>
    <button type="button" hx-get="/transactions/{{.ID}}/view" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint border border-[#EAE8E4] rounded-md px-[7px] py-[3px]">Hủy</button>
  </span>
  <input name="amount" type="number" min="1" value="{{.Amount}}" required class="w-[132px] shrink-0 text-right font-mono text-[14px] border border-border-input rounded px-2 h-8">
</form>
{{end}}

{{define "transaction_row_delete_confirm"}}
<div id="transaction-row-{{.ID}}" class="flex items-center justify-between px-4 py-[13px] border-b border-border-list last:border-b-0" style="background-color:#FEF7F5">
  <span class="text-[13px] text-ink">Xóa giao dịch này?</span>
  <span class="flex items-center gap-2">
    <button type="button" hx-delete="/transactions/{{.ID}}" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="px-3 h-8 rounded-lg bg-expense text-white text-[12px] font-semibold">Xóa</button>
    <button type="button" hx-get="/transactions/{{.ID}}/view" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="px-3 h-8 rounded-lg border border-border-input text-[12px] text-ink">Hủy</button>
  </span>
</div>
{{end}}
```

- [ ] **Step 6: Add the new handlers to `internal/handlers/transaction_handlers.go` and rewrite `deleteTransactionHandler`**

Add `"log"` to the imports. Append these functions, and replace `deleteTransactionHandler` entirely:

```go
// dateInputValue formats a DATE column for an <input type="date"> value
// attribute (always "2006-01-02", regardless of display locale) -- distinct
// from dateFull/dateShort, which are for read-only display.
func dateInputValue(d pgtype.Date) string {
	return d.Time.Format("2006-01-02")
}

func editTransactionRowHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		txn, err := deps.Queries.GetTransaction(r.Context(), sqlcgen.GetTransactionParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		var options []sqlcgen.Category
		for _, c := range allCategories {
			if c.Type == txn.Type {
				options = append(options, c)
			}
		}

		renderNamed(w, r, deps, "transactions", "transaction_row_edit", "", map[string]any{
			"ID": txn.ID, "CategoryID": txn.CategoryID, "Description": txn.Description,
			"Amount": txn.Amount, "OccurredOnValue": dateInputValue(txn.OccurredOn),
			"CategoryOptions": options,
		})
	}
}

func viewTransactionRowHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		txn, err := deps.Queries.GetTransactionWithCategory(r.Context(), sqlcgen.GetTransactionWithCategoryParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		renderNamed(w, r, deps, "transactions", "transaction_row", "", map[string]any{
			"ID": txn.ID, "CategoryName": txn.CategoryName, "CategoryColor": txn.CategoryColor,
			"Description": txn.Description, "OccurredOn": txn.OccurredOn, "Amount": txn.Amount, "Type": txn.Type,
		})
	}
}

func deleteConfirmTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if _, err := deps.Queries.GetTransaction(r.Context(), sqlcgen.GetTransactionParams{ID: id, UserID: userID}); err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}
		renderNamed(w, r, deps, "transactions", "transaction_row_delete_confirm", "", map[string]any{"ID": id})
	}
}

func updateTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		existing, err := deps.Queries.GetTransaction(r.Context(), sqlcgen.GetTransactionParams{ID: id, UserID: userID})
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid category", http.StatusBadRequest)
			return
		}
		amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
		if err != nil || amount <= 0 {
			http.Error(w, "invalid amount", http.StatusBadRequest)
			return
		}
		occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
		if err != nil {
			http.Error(w, "invalid date", http.StatusBadRequest)
			return
		}
		description := r.FormValue("description")

		category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{ID: categoryID, UserID: pgInt64(userID)})
		if err != nil {
			http.Error(w, "category not found", http.StatusForbidden)
			return
		}

		var formErr string
		switch {
		case category.Type != existing.Type:
			formErr = "Loại giao dịch không khớp với danh mục đã chọn"
		case len([]rune(description)) > 200:
			formErr = "Ghi chú tối đa 200 ký tự"
		case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
			formErr = "Ngày giao dịch không được ở quá xa trong tương lai"
		}
		if formErr != "" {
			allCategories, _ := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
			var options []sqlcgen.Category
			for _, c := range allCategories {
				if c.Type == existing.Type {
					options = append(options, c)
				}
			}
			renderNamed(w, r, deps, "transactions", "transaction_row_edit", "", map[string]any{
				"ID": id, "CategoryID": categoryID, "Description": description, "Amount": amount,
				"OccurredOnValue": r.FormValue("occurred_on"), "CategoryOptions": options, "Error": formErr,
			})
			return
		}

		updated, err := deps.Queries.UpdateTransaction(r.Context(), sqlcgen.UpdateTransactionParams{
			ID: id, UserID: userID, CategoryID: categoryID, Amount: amount, Type: existing.Type,
			Description: description, OccurredOn: pgDate(occurredOn),
		})
		if err != nil {
			log.Printf("update transaction: %v", err)
			http.Error(w, "could not update transaction", http.StatusInternalServerError)
			return
		}

		from, to := currentMonthRange()
		totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			http.Error(w, "could not load totals", http.StatusInternalServerError)
			return
		}
		transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			http.Error(w, "could not load transactions", http.StatusInternalServerError)
			return
		}

		// Reuses transaction_create_response (Task 6) -- despite the name it
		// is just "a row plus the three OOB total fragments", which is
		// exactly what a successful edit needs too.
		renderNamed(w, r, deps, "transactions", "transaction_create_response", "", map[string]any{
			"Row": map[string]any{
				"ID": updated.ID, "CategoryName": category.Name, "CategoryColor": category.Color,
				"Description": updated.Description, "OccurredOn": updated.OccurredOn,
				"Amount": updated.Amount, "Type": updated.Type,
			},
			"Totals": map[string]any{
				"TotalExpense": totals.TotalExpense, "TotalIncome": totals.TotalIncome,
				"Remaining": totals.TotalIncome - totals.TotalExpense, "Count": len(transactions),
			},
		})
	}
}

func deleteTransactionHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		id, err := strconv.ParseInt(chiURLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if _, err := deps.Queries.DeleteTransaction(r.Context(), sqlcgen.DeleteTransactionParams{ID: id, UserID: userID}); err != nil {
			http.Error(w, "could not delete transaction", http.StatusInternalServerError)
			return
		}

		from, to := currentMonthRange()
		totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			http.Error(w, "could not load totals", http.StatusInternalServerError)
			return
		}
		transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			http.Error(w, "could not load transactions", http.StatusInternalServerError)
			return
		}

		renderNamed(w, r, deps, "transactions", "totals_oob", "", map[string]any{
			"TotalExpense": totals.TotalExpense, "TotalIncome": totals.TotalIncome,
			"Remaining": totals.TotalIncome - totals.TotalExpense, "Count": len(transactions),
		})
	}
}
```

- [ ] **Step 7: Update `internal/handlers/router.go`**

Replace:

```go
pr.Post("/transactions/{id}/delete", deleteTransactionHandler(deps))
```

with:

```go
pr.Get("/transactions/{id}/edit", editTransactionRowHandler(deps))
pr.Get("/transactions/{id}/view", viewTransactionRowHandler(deps))
pr.Get("/transactions/{id}/delete-confirm", deleteConfirmTransactionHandler(deps))
pr.Patch("/transactions/{id}", updateTransactionHandler(deps))
pr.Delete("/transactions/{id}", deleteTransactionHandler(deps))
```

- [ ] **Step 8: Run the full test suite to verify it passes**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

- [ ] **Step 9: Manual smoke check**

Run: `go run ./cmd/server`, visit `/transactions`, hover a row: click "Sửa", change the amount, click "Lưu", confirm the row updates in place and totals adjust; click "Xóa" on another row, confirm the pink confirm bar appears, click "Hủy" to confirm it reverts, then click "Xóa" again and confirm to confirm the row disappears and totals adjust.

---

### Task 9: Giao dịch — mobile layout (stat cards, FAB, bottom-sheet add form)

**Files:**
- Rewrite: `internal/web/templates/transactions.html` (supersedes Task 7's version)
- Rewrite: `internal/web/templates/transaction_row.html` (supersedes Task 8's version — adds mobile row markup, extends `totals_oob`)
- Modify: `internal/handlers/transaction_handlers.go` (new `categoryChipsHandler`, `ui_source`-aware error retargeting, `HX-Trigger` on success)
- Modify: `internal/handlers/router.go`
- Modify: `internal/handlers/transaction_handlers_test.go`

**Interfaces:**
- `categoryChipsHandler(deps Deps) http.HandlerFunc` — `GET /transactions/category-chips?type=expense|income`, the mobile bottom sheet's equivalent of `categoryOptionsHandler`.
- `handleCreateTransaction` now reads a `ui_source` hidden field (`"desktop"` or `"mobile"`) to decide which form fragment/id to retarget on validation failure, and sets `HX-Trigger: transaction-created` on success.

Design notes:
- **A latent bug from Task 6 gets fixed here first**: the desktop quick-add form's field-reset ran on `hx-on::after-request` checking `event.detail.successful`, which is `true` for *any* 2xx response — including the 200-with-`HX-Retarget` validation-error response. That means a failed submission was silently clearing the amount/description fields the user just typed. Fixing it properly (an explicit `HX-Trigger: transaction-created` header, fired only on the real success path, listened for via `hx-on:transaction-created` instead of the generic request-lifecycle event) is what makes the mobile sheet's "close on success, stay open on error" behavior possible at all, so both forms move to this pattern together.
- **Two independent `<form>` elements post to the same `POST /transactions`** — one desktop (`#quick-add-form`, unchanged fields), one mobile (`#mobile-quick-add-form`, chips instead of a `<select>`, a large `inputmode="numeric"` amount field). Unlike the category add-form (Task 4), these aren't the same markup relocated between breakpoints — SPEC.md gives them genuinely different field widgets — so both simply exist in the DOM, one hidden via `md:hidden`/`hidden md:flex` at a time. Each form carries a hidden `ui_source` field so the handler retargets the correct one on error.
- **Mobile row actions use the "nhấn giữ mở action sheet" alternative SPEC.md itself offers**, not swipe-to-reveal — SPEC.md section 3.4 literally says "vuốt trái ... hoặc nhấn giữ mở action sheet", so a tap-triggered (not press-and-hold, per the Global Constraints' documented simplification) "⋯" button opening a small action sheet is already one of the two spec'd options, just substituting tap for long-press.
- **Mobile inline-edit reuses `transaction_row_edit` as-is** (Task 8), just with `flex-wrap` added so it degrades to two lines on a narrow viewport instead of overflowing, rather than building a third, mobile-specific edit form. If Task 12's visual QA pass finds this reads badly, revisit then.
- **`totals_oob` (Task 6) gains a fourth OOB block**, `#mobile-stat-cards`, so the two mobile stat cards stay in sync with every create/update/delete exactly like the desktop totals row already does.

- [ ] **Step 1: Add a mobile-create test to `internal/handlers/transaction_handlers_test.go`**

```go
func TestCreateTransactionViaMobileFormTriggersHXTrigger(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-mobile-create@example.com", "s3cret-pass")
	ctx := context.Background()

	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	user, err := deps.Queries.GetUserByEmail(ctx, "txn-mobile-create@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"15000"},
		"type":        {"expense"},
		"occurred_on": {time.Now().Format("2006-01-02")},
		"ui_source":   {"mobile"},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Trigger"); got != "transaction-created" {
		t.Fatalf("expected HX-Trigger: transaction-created, got %q", got)
	}
}

func TestCreateTransactionValidationErrorRetargetsMobileForm(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "txn-mobile-error@example.com", "s3cret-pass")

	categories, err := deps.Queries.ListCategoriesForUser(context.Background(), pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(category.ID, 10)},
		"amount":      {"15000"},
		"type":        {"income"}, // mismatched on purpose
		"occurred_on": {time.Now().Format("2006-01-02")},
		"ui_source":   {"mobile"},
	}
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("HX-Retarget"); got != "#mobile-quick-add-form" {
		t.Fatalf("expected HX-Retarget: #mobile-quick-add-form, got %q", got)
	}
	if rec.Header().Get("HX-Trigger") != "" {
		t.Fatal("expected no HX-Trigger on a validation-error response")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestCreateTransactionViaMobileForm -run TestCreateTransactionValidationErrorRetargetsMobileForm`
Expected: FAIL — `ui_source`, `HX-Trigger`, and the mobile retarget id don't exist yet.

- [ ] **Step 3: Rewrite `internal/web/templates/transaction_row.html`**

```html
{{define "transaction_row"}}
<div id="transaction-row-{{.ID}}" class="group flex items-center gap-3 md:gap-4 px-4 py-3 md:py-[13px] border-b border-border-list last:border-b-0 hover:bg-surface-alt">
  <span class="w-2 h-2 rounded-full shrink-0 md:hidden" style="background-color: {{.CategoryColor}}"></span>
  <span class="hidden md:inline w-[46px] shrink-0 font-mono text-[12px] text-ink-faintest">{{dateShort .OccurredOn}}</span>
  <span class="hidden md:flex w-[150px] shrink-0 items-center gap-2 text-[13px] font-medium text-ink">
    <span class="w-2 h-2 rounded-full shrink-0" style="background-color: {{.CategoryColor}}"></span>
    <span class="truncate">{{.CategoryName}}</span>
  </span>
  <span class="flex-1 min-w-0 hidden md:block text-[13px] text-ink-faint truncate">{{.Description}}</span>
  <span class="flex-1 min-w-0 md:hidden flex flex-col">
    <span class="text-[14px] font-medium text-ink truncate">{{.CategoryName}}</span>
    <span class="text-[12px] text-ink-faintest truncate">{{dateShort .OccurredOn}}{{if .Description}} · {{.Description}}{{end}}</span>
  </span>
  <span class="hidden md:flex shrink-0 items-center gap-1 opacity-0 group-hover:opacity-100">
    <button type="button" hx-get="/transactions/{{.ID}}/edit" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint border border-[#EAE8E4] rounded-md px-[7px] py-[3px]">Sửa</button>
    <button type="button" hx-get="/transactions/{{.ID}}/delete-confirm" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint border border-[#EAE8E4] rounded-md px-[7px] py-[3px]">Xóa</button>
  </span>
  <span class="shrink-0 text-right font-mono text-[14px] font-medium {{if eq .Type "expense"}}text-expense{{else}}text-income{{end}}">{{vndSigned .Amount .Type}}</span>
  <button type="button" onclick="document.getElementById('mobile-actions-{{.ID}}').showModal()" class="md:hidden text-ink-faint text-[16px] px-1 leading-none">⋯</button>
  <dialog id="mobile-actions-{{.ID}}" class="md:hidden rounded-t-[16px] w-full max-w-full m-0 mt-auto mb-0 p-0 backdrop:bg-black/32">
    <div class="p-4 flex flex-col gap-1">
      <button type="button" onclick="document.getElementById('mobile-actions-{{.ID}}').close()" hx-get="/transactions/{{.ID}}/edit" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-left px-3 py-3 text-[14px] text-ink rounded-lg hover:bg-track">Sửa</button>
      <button type="button" onclick="document.getElementById('mobile-actions-{{.ID}}').close()" hx-get="/transactions/{{.ID}}/delete-confirm" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-left px-3 py-3 text-[14px] text-expense rounded-lg hover:bg-track">Xóa</button>
    </div>
  </dialog>
</div>
{{end}}

{{define "totals_oob"}}
<div id="totals-summary" hx-swap-oob="true" class="flex items-center gap-[18px]">
  <span class="text-[13px] text-ink-faint">Chi <span class="font-mono font-medium text-expense">{{vnd .TotalExpense}}</span></span>
  <span class="text-[13px] text-ink-faint">Thu <span class="font-mono font-medium text-income">{{vnd .TotalIncome}}</span></span>
</div>
<span id="transaction-count" hx-swap-oob="true" class="text-[12px] text-ink-faint">{{.Count}} giao dịch</span>
<div id="remaining-row" hx-swap-oob="true" class="flex items-center justify-between px-4 py-[13px] bg-surface-alt">
  <span class="text-[13px] font-semibold text-ink">Còn lại tháng này</span>
  <span class="w-[132px] text-right font-mono text-[15px] font-semibold {{if lt .Remaining 0}}text-expense{{else}}text-ink{{end}}">{{vndBalance .Remaining}}</span>
</div>
<div id="mobile-stat-cards" hx-swap-oob="true" class="md:hidden flex items-center gap-3">
  <div class="flex-1 rounded-[10px] bg-surface border border-border-card px-3 py-[10px]">
    <p class="text-[11px] text-ink-faint">Chi</p>
    <p class="font-mono text-[15px] font-semibold text-expense">{{vnd .TotalExpense}}</p>
  </div>
  <div class="flex-1 rounded-[10px] bg-surface border border-border-card px-3 py-[10px]">
    <p class="text-[11px] text-ink-faint">Thu</p>
    <p class="font-mono text-[15px] font-semibold text-income">{{vnd .TotalIncome}}</p>
  </div>
</div>
{{end}}

{{define "transaction_create_response"}}{{template "transaction_row" .Row}}{{template "totals_oob" .Totals}}{{end}}

{{define "transaction_row_edit"}}
<form id="transaction-row-{{.ID}}" hx-patch="/transactions/{{.ID}}" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="flex items-center gap-4 px-4 py-[10px] border-b border-border-list last:border-b-0 bg-surface-alt flex-wrap">
  <input name="occurred_on" type="date" value="{{.OccurredOnValue}}" required class="w-[46px] shrink-0 font-mono text-[11px] border border-border-input rounded px-1 h-8">
  <select name="category_id" class="w-[150px] shrink-0 text-[13px] border border-border-input rounded px-1 h-8">
    {{range .CategoryOptions}}<option value="{{.ID}}" {{if eq .ID $.CategoryID}}selected{{end}}>{{.Name}}</option>{{end}}
  </select>
  <input name="description" type="text" value="{{.Description}}" class="flex-1 min-w-0 text-[13px] border border-border-input rounded px-2 h-8">
  {{if .Error}}<span class="text-[12px] text-expense shrink-0">{{.Error}}</span>{{end}}
  <span class="shrink-0 flex items-center gap-1">
    <button type="submit" class="text-[12px] text-accent font-semibold border border-border-input rounded-md px-[7px] py-[3px]">Lưu</button>
    <button type="button" hx-get="/transactions/{{.ID}}/view" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="text-[12px] text-ink-faint border border-[#EAE8E4] rounded-md px-[7px] py-[3px]">Hủy</button>
  </span>
  <input name="amount" type="number" min="1" value="{{.Amount}}" required class="w-[132px] shrink-0 text-right font-mono text-[14px] border border-border-input rounded px-2 h-8">
</form>
{{end}}

{{define "transaction_row_delete_confirm"}}
<div id="transaction-row-{{.ID}}" class="flex items-center justify-between px-4 py-[13px] border-b border-border-list last:border-b-0" style="background-color:#FEF7F5">
  <span class="text-[13px] text-ink">Xóa giao dịch này?</span>
  <span class="flex items-center gap-2">
    <button type="button" hx-delete="/transactions/{{.ID}}" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="px-3 h-8 rounded-lg bg-expense text-white text-[12px] font-semibold">Xóa</button>
    <button type="button" hx-get="/transactions/{{.ID}}/view" hx-target="#transaction-row-{{.ID}}" hx-swap="outerHTML" class="px-3 h-8 rounded-lg border border-border-input text-[12px] text-ink">Hủy</button>
  </span>
</div>
{{end}}

{{define "category_chips"}}
<div id="mobile-category-chips" class="flex flex-wrap gap-2">
  {{range $i, $c := .Categories}}
  <label class="px-[13px] py-[9px] rounded-full border flex items-center gap-2 cursor-pointer {{if eq $i 0}}border-accent bg-accent/[0.08]{{else}}border-border-input{{end}} has-[:checked]:border-accent has-[:checked]:bg-accent/[0.08]">
    <input type="radio" name="category_id" value="{{$c.ID}}" {{if eq $i 0}}checked{{end}} class="hidden">
    <span class="w-2 h-2 rounded-full" style="background-color: {{$c.Color}}"></span>
    <span class="text-[13px]">{{$c.Name}}</span>
  </label>
  {{end}}
</div>
{{end}}
```

- [ ] **Step 4: Rewrite `internal/web/templates/transactions.html`**

```html
{{define "content"}}
<div class="max-w-[880px] mx-auto px-5 md:px-6 pt-4 md:pt-7 pb-24 md:pb-9 flex flex-col gap-[14px] md:gap-[18px]">
  <div class="hidden md:block" id="quick-add-form-wrapper">{{template "quick_add_form" .}}</div>
  {{template "transactions_month_section" .}}
</div>

<button type="button" onclick="document.getElementById('mobile-add-sheet').showModal()"
  class="md:hidden fixed z-30 h-12 px-5 rounded-full bg-accent text-white text-[14px] font-semibold flex items-center gap-1"
  style="right:16px; bottom:74px; box-shadow:0 8px 20px -6px rgba(59,110,207,0.6);">＋ Thêm</button>

<dialog id="mobile-add-sheet" class="md:hidden w-full max-w-full m-0 mt-auto mb-0 rounded-t-[20px] rounded-b-none p-0 backdrop:bg-black/32">
  <div class="p-[18px] pb-7">
    <div class="w-[38px] h-1 rounded-full bg-border-input mx-auto mb-4"></div>
    <p class="text-[17px] font-semibold text-ink mb-4">Thêm giao dịch</p>
    {{template "mobile_quick_add_form" .}}
  </div>
</dialog>
{{end}}

{{define "mobile_quick_add_form"}}
<form id="mobile-quick-add-form" hx-post="/transactions" hx-target="#transaction-list" hx-swap="afterbegin"
  hx-on:transaction-created="document.getElementById('mobile-add-sheet').close(); this.reset();"
  class="flex flex-col gap-4">
  <input type="hidden" name="ui_source" value="mobile">
  <div class="flex bg-track rounded-lg p-[3px] gap-1 h-10">
    <label class="flex-1 flex items-center justify-center cursor-pointer">
      <input type="radio" name="type" value="expense" {{if ne .SelectedType "income"}}checked{{end}} class="peer hidden"
        hx-get="/transactions/category-chips" hx-target="#mobile-category-chips" hx-trigger="change" hx-vals='{"type":"expense"}'>
      <span class="text-[14px] peer-checked:font-semibold">Chi</span>
    </label>
    <label class="flex-1 flex items-center justify-center cursor-pointer">
      <input type="radio" name="type" value="income" {{if eq .SelectedType "income"}}checked{{end}} class="peer hidden"
        hx-get="/transactions/category-chips" hx-target="#mobile-category-chips" hx-trigger="change" hx-vals='{"type":"income"}'>
      <span class="text-[14px] peer-checked:font-semibold">Thu</span>
    </label>
  </div>
  <input name="amount" type="number" inputmode="numeric" min="1" required placeholder="0"
    class="h-14 px-4 rounded-[10px] border-2 border-accent font-mono text-[24px] font-semibold text-center focus:outline-none focus:ring-[3px] focus:ring-accent/[0.12]">
  {{template "category_chips" .}}
  <div class="flex gap-3">
    <input name="occurred_on" type="date" value="{{.Today}}" required class="flex-1 h-12 px-3 rounded-[10px] border border-border-input font-mono text-[14px]">
    <input name="description" type="text" placeholder="Ghi chú" class="flex-1 h-12 px-3 rounded-[10px] border border-border-input text-[14px]">
  </div>
  {{if .QuickAddError}}<p class="text-[12px] text-expense">{{.QuickAddError}}</p>{{end}}
  <button type="submit" class="h-[50px] rounded-[10px] bg-accent text-white text-[14px] font-semibold">Lưu giao dịch</button>
</form>
{{end}}

{{define "transactions_month_section"}}
<div id="transactions-month-section" class="flex flex-col gap-[14px] md:gap-[18px]">
  <div class="flex items-center justify-between">
    <h1 class="md:hidden text-[19px] font-semibold text-ink">Giao dịch</h1>
    <div class="hidden md:flex items-center gap-2 relative">
      <button type="button" onclick="this.nextElementSibling.classList.toggle('hidden')" class="h-8 px-3 rounded-lg border border-border-input bg-white text-[13px] font-medium text-ink flex items-center gap-1">{{.MonthLabel}} <span class="text-placeholder">▾</span></button>
      <div class="hidden absolute left-0 top-9 z-20 bg-white border border-border-card rounded-lg py-1 w-[160px] shadow-sm">
        <button type="button" hx-get="/transactions?thang={{.CurrentMonthValue}}" hx-target="#transactions-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">Tháng này</button>
        {{range .AvailableMonths}}
        <button type="button" hx-get="/transactions?thang={{.Value}}" hx-target="#transactions-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">{{.Label}}</button>
        {{end}}
      </div>
      <span id="transaction-count" class="text-[12px] text-ink-faint">{{len .Transactions}} giao dịch</span>
    </div>
    <div class="md:hidden relative">
      <button type="button" onclick="this.nextElementSibling.classList.toggle('hidden')" class="h-8 px-3 rounded-lg border border-border-input bg-white text-[13px] font-medium text-ink flex items-center gap-1">{{.MonthLabel}} <span class="text-placeholder">▾</span></button>
      <div class="hidden absolute right-0 top-9 z-20 bg-white border border-border-card rounded-lg py-1 w-[160px] shadow-sm">
        <button type="button" hx-get="/transactions?thang={{.CurrentMonthValue}}" hx-target="#transactions-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">Tháng này</button>
        {{range .AvailableMonths}}
        <button type="button" hx-get="/transactions?thang={{.Value}}" hx-target="#transactions-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">{{.Label}}</button>
        {{end}}
      </div>
    </div>
    <div id="totals-summary" class="hidden md:flex items-center gap-[18px]">
      <span class="text-[13px] text-ink-faint">Chi <span class="font-mono font-medium text-expense">{{vnd .TotalExpense}}</span></span>
      <span class="text-[13px] text-ink-faint">Thu <span class="font-mono font-medium text-income">{{vnd .TotalIncome}}</span></span>
    </div>
  </div>

  <div id="mobile-stat-cards" class="md:hidden flex items-center gap-3">
    <div class="flex-1 rounded-[10px] bg-surface border border-border-card px-3 py-[10px]">
      <p class="text-[11px] text-ink-faint">Chi</p>
      <p class="font-mono text-[15px] font-semibold text-expense">{{vnd .TotalExpense}}</p>
    </div>
    <div class="flex-1 rounded-[10px] bg-surface border border-border-card px-3 py-[10px]">
      <p class="text-[11px] text-ink-faint">Thu</p>
      <p class="font-mono text-[15px] font-semibold text-income">{{vnd .TotalIncome}}</p>
    </div>
  </div>

  <div class="bg-surface border border-border-card rounded-xl overflow-hidden">
    <div id="transaction-list">
      {{range .Transactions}}{{template "transaction_row" .}}{{end}}
    </div>
    {{if not .Transactions}}
    <div class="text-center py-11 px-7">
      <p class="text-[15px] font-semibold text-ink">Chưa có giao dịch nào trong {{.MonthLabelLower}}</p>
      <p class="text-[13px] text-ink-faint mt-1 max-w-[300px] mx-auto">Thêm giao dịch đầu tiên bằng form phía trên, hoặc chọn tháng khác để xem lại.</p>
    </div>
    {{end}}
    <div id="remaining-row" class="flex items-center justify-between px-4 py-[13px] bg-surface-alt">
      <span class="text-[13px] font-semibold text-ink">Còn lại tháng này</span>
      <span class="w-[132px] text-right font-mono text-[15px] font-semibold {{if lt .Remaining 0}}text-expense{{else}}text-ink{{end}}">{{vndBalance .Remaining}}</span>
    </div>
  </div>
</div>
{{end}}

{{define "quick_add_form"}}
<div class="bg-surface border border-border-card rounded-xl p-4">
  <form id="quick-add-form" hx-post="/transactions" hx-target="#transaction-list" hx-swap="afterbegin"
    hx-on:transaction-created="this.querySelector('[name=amount]').value=''; this.querySelector('[name=description]').value='';"
    class="flex items-end gap-[10px] flex-wrap">
    <input type="hidden" name="ui_source" value="desktop">
    <div class="flex flex-col gap-[6px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Loại</label>
      <div class="flex bg-track rounded-lg p-[3px] gap-1 h-9">
        <label class="px-3 flex items-center cursor-pointer">
          <input type="radio" name="type" value="expense" {{if ne .SelectedType "income"}}checked{{end}} class="peer hidden"
            hx-get="/transactions/category-options" hx-target="#category-select" hx-trigger="change" hx-vals='{"type":"expense"}'>
          <span class="text-[13px] peer-checked:font-semibold">Chi</span>
        </label>
        <label class="px-3 flex items-center cursor-pointer">
          <input type="radio" name="type" value="income" {{if eq .SelectedType "income"}}checked{{end}} class="peer hidden"
            hx-get="/transactions/category-options" hx-target="#category-select" hx-trigger="change" hx-vals='{"type":"income"}'>
          <span class="text-[13px] peer-checked:font-semibold">Thu</span>
        </label>
      </div>
    </div>
    <div class="flex flex-col gap-[6px] w-[168px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Danh mục</label>
      {{template "category_options" .}}
    </div>
    <div class="flex flex-col gap-[6px] w-[150px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Số tiền</label>
      <input name="amount" type="number" min="1" required class="h-9 px-3 rounded-lg border border-border-input font-mono text-[14px] font-medium">
    </div>
    <div class="flex flex-col gap-[6px] w-[128px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Ngày</label>
      <input name="occurred_on" type="date" value="{{.Today}}" required class="h-9 px-3 rounded-lg border border-border-input font-mono text-[13px]">
    </div>
    <div class="flex flex-col gap-[6px] flex-1 min-w-[140px]">
      <label class="text-[11px] font-semibold uppercase tracking-wide text-ink-faint">Ghi chú</label>
      <input name="description" type="text" placeholder="Không bắt buộc" class="h-9 px-3 rounded-lg border border-border-input text-[13px]">
    </div>
    <button type="submit" class="h-9 px-[18px] rounded-lg bg-accent text-white text-[13px] font-semibold">Thêm</button>
  </form>
  {{if .QuickAddError}}<p class="text-[12px] text-expense mt-2">{{.QuickAddError}}</p>{{end}}
</div>
{{end}}

{{define "category_options"}}
<select id="category-select" name="category_id" class="h-9 px-3 rounded-lg border border-border-input text-[13px] bg-white">
  {{range .Categories}}<option value="{{.ID}}" style="color: {{.Color}}">● {{.Name}}</option>{{end}}
</select>
{{end}}
```

- [ ] **Step 5: Add `categoryChipsHandler` and update `handleCreateTransaction` in `internal/handlers/transaction_handlers.go`**

```go
func categoryChipsHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())
		typ := r.FormValue("type")
		if typ != "income" {
			typ = "expense"
		}
		categories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}
		var filtered []sqlcgen.Category
		for _, c := range categories {
			if c.Type == typ {
				filtered = append(filtered, c)
			}
		}
		renderNamed(w, r, deps, "transactions", "category_chips", "", map[string]any{"Categories": filtered})
	}
}
```

Replace `handleCreateTransaction`'s two `w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")` / `HX-Reswap` / `renderTransactionsPageForm(...)` call sites with a small `retarget` closure that branches on `ui_source`, and set `HX-Trigger` right before the success render:

```go
func handleCreateTransaction(w http.ResponseWriter, r *http.Request, deps Deps, userID int64) {
	categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid category", http.StatusBadRequest)
		return
	}
	amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if err != nil || amount <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}
	occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}
	txnType := r.FormValue("type")
	if txnType != "expense" && txnType != "income" {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	source := r.FormValue("ui_source")

	retarget := func(errMsg string) {
		if source == "mobile" {
			w.Header().Set("HX-Retarget", "#mobile-quick-add-form")
			w.Header().Set("HX-Reswap", "outerHTML")
			renderTransactionsPageMobileForm(w, r, deps, userID, errMsg, txnType)
			return
		}
		w.Header().Set("HX-Retarget", "#quick-add-form-wrapper")
		w.Header().Set("HX-Reswap", "outerHTML")
		renderTransactionsPageForm(w, r, deps, userID, errMsg, txnType)
	}

	category, err := deps.Queries.GetCategoryForUser(r.Context(), sqlcgen.GetCategoryForUserParams{
		ID: categoryID, UserID: pgInt64(userID),
	})
	if err != nil {
		http.Error(w, "category not found", http.StatusForbidden)
		return
	}

	var formErr string
	switch {
	case category.Type != txnType:
		formErr = "Loại giao dịch không khớp với danh mục đã chọn"
	case len([]rune(r.FormValue("description"))) > 200:
		formErr = "Ghi chú tối đa 200 ký tự"
	case occurredOn.After(time.Now().In(vietnamLocation).AddDate(0, 0, 7)):
		formErr = "Ngày giao dịch không được ở quá xa trong tương lai"
	}
	if formErr != "" {
		retarget(formErr)
		return
	}

	created, err := deps.Queries.CreateTransaction(r.Context(), sqlcgen.CreateTransactionParams{
		UserID: userID, CategoryID: categoryID, Amount: amount, Type: txnType,
		Description: r.FormValue("description"), OccurredOn: pgDate(occurredOn),
	})
	if err != nil {
		retarget("Không thể thêm giao dịch, vui lòng thử lại.")
		return
	}

	from, to := currentMonthRange()
	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load totals", http.StatusInternalServerError)
		return
	}
	transactions, err := deps.Queries.ListTransactionsForMonth(r.Context(), sqlcgen.ListTransactionsForMonthParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "transaction-created")
	renderNamed(w, r, deps, "transactions", "transaction_create_response", "", map[string]any{
		"Row": map[string]any{
			"ID": created.ID, "CategoryName": category.Name, "CategoryColor": category.Color,
			"Description": created.Description, "OccurredOn": created.OccurredOn,
			"Amount": created.Amount, "Type": created.Type,
		},
		"Totals": map[string]any{
			"TotalExpense": totals.TotalExpense, "TotalIncome": totals.TotalIncome,
			"Remaining": totals.TotalIncome - totals.TotalExpense, "Count": len(transactions),
		},
	})
}

// renderTransactionsPageMobileForm mirrors renderTransactionsPageForm but
// re-renders the mobile bottom-sheet form fragment instead of the desktop
// one, for handleCreateTransaction's ui_source == "mobile" error path.
func renderTransactionsPageMobileForm(w http.ResponseWriter, r *http.Request, deps Deps, userID int64, errMsg, selectedType string) {
	allCategories, err := deps.Queries.ListCategoriesForUser(r.Context(), pgInt64(userID))
	if err != nil {
		http.Error(w, "could not load categories", http.StatusInternalServerError)
		return
	}
	var filteredCategories []sqlcgen.Category
	for _, c := range allCategories {
		if c.Type == selectedType {
			filteredCategories = append(filteredCategories, c)
		}
	}
	renderNamed(w, r, deps, "transactions", "mobile_quick_add_form", "", map[string]any{
		"Categories":    filteredCategories,
		"SelectedType":  selectedType,
		"Today":         time.Now().In(vietnamLocation).Format("2006-01-02"),
		"QuickAddError": errMsg,
	})
}
```

- [ ] **Step 6: Add `GET /transactions/category-chips` to `internal/handlers/router.go`**

```go
pr.Get("/transactions/category-chips", categoryChipsHandler(deps))
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestCreateTransactionViaMobileForm -run TestCreateTransactionValidationErrorRetargetsMobileForm`
Expected: PASS.

- [ ] **Step 8: Run the full test suite**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

- [ ] **Step 9: Manual smoke check**

Run: `go run ./cmd/server`, resize the browser to ~390px (or use devtools device mode), visit `/transactions`: confirm the two stat cards, FAB, and bottom sheet render; open the sheet, switch Loại, confirm chips reload; submit a valid transaction and confirm the sheet closes and the row appears; submit an invalid one (e.g. a mismatched category) and confirm the sheet stays open showing the error instead of closing.

---

### Task 10: Tổng quan — queries and handler (4-month series, prev-month comparison)

**Files:**
- Rewrite: `internal/handlers/report_handlers.go`
- Create: `internal/handlers/report_handlers_internal_test.go`

**Interfaces:**
- `comparisonText(current, previous int64, hasPrevData bool) string`
- `buildPieData(breakdown []sqlcgen.CategoryBreakdownRow, totalExpense int64) (labels []string, values []int64, colors []string, legend []pieLegendEntry)`
- `buildBarSeries(series []sqlcgen.MonthlyTotalsSeriesRow, currentMonthStart time.Time, months int) (labels []string, chi []int64, thu []int64)`

Design notes:
- **These three functions are pure and get their own unit tests** (`report_handlers_internal_test.go`, package `handlers` like `pg_internal_test.go`) rather than being exercised only through an HTTP round-trip. That matters specifically for this task: Task 10 only touches the handler, not `dashboard.html` (Task 11's job), so a body-content assertion against the still-old dashboard template couldn't see any of this task's new output anyway — the old template only prints `.TotalExpense`/`.TotalIncome` as raw numbers, which this task's data map still provides under the same keys, so the two existing dashboard tests (`TestDashboardShowsMonthlyTotal`, `TestEndToEndRegisterAddTransactionSeeDashboard`) keep passing unmodified through this task.
- **Pie chart top-N**: SPEC.md section 5 caps the legend at the 6 largest expense categories, aggregating the rest into a synthetic `"Khác"` slice colored `#A1A1AA` (the same reserved gray as the real "Khác" default category — SPEC.md reuses it for this synthetic aggregate too).
- **Bar chart is always exactly `barMonths` (4) points**, oldest to newest, zero-padding any month `MonthlyTotalsSeries` didn't return a row for (a month with zero transactions is indistinguishable from a month never queried, at the SQL level — `GROUP BY` simply omits it).
- **Previous-month "no data" vs "zero" are different things**: `hasPrevData` is true whenever the previous month had *any* transaction (expense or income), even if the specific metric being compared (e.g. expense) happened to be exactly `0` that month — that 0 is a real answer ("bạn không chi gì tháng trước"), not a missing one. Only a previous month with zero transactions at all shows SPEC.md's "Chưa có dữ liệu tháng trước".
- **SPEC.md's empty-state wording is internally ambiguous** — it says both "vùng hai biểu đồ thay bằng một card" (both chart areas become one empty-state card) and, two lines later, that the bar chart still draws if *any* of the 4 months has data. This task computes both `CurrentMonthEmpty` (this month has zero transactions) and `HasAnyMonthData` (across all 4 bar-chart months) so Task 11's template can implement the read that reconciles them: a combined empty-state card only when *neither* is true; otherwise the pie chart alone is replaced and the bar chart still renders.

- [ ] **Step 1: Write `internal/handlers/report_handlers_internal_test.go`**

```go
package handlers

import (
	"testing"
	"time"

	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestComparisonText(t *testing.T) {
	if got := comparisonText(0, 0, false); got != "Chưa có dữ liệu tháng trước" {
		t.Errorf("comparisonText(no prev data) = %q", got)
	}
	if got := comparisonText(8800000, 10000000, true); got != "Tháng trước 10.000.000₫ · giảm 12%" {
		t.Errorf("comparisonText(decrease) = %q", got)
	}
	if got := comparisonText(11000000, 10000000, true); got != "Tháng trước 10.000.000₫ · tăng 10%" {
		t.Errorf("comparisonText(increase) = %q", got)
	}
}

func TestBuildPieDataCapsAtSixPlusKhac(t *testing.T) {
	var breakdown []sqlcgen.CategoryBreakdownRow
	for i := 0; i < 8; i++ {
		breakdown = append(breakdown, sqlcgen.CategoryBreakdownRow{
			CategoryName: "Cat", CategoryColor: "#D97757", Total: int64(100 - i),
		})
	}
	labels, values, colors, legend := buildPieData(breakdown, 700)
	if len(labels) != 7 {
		t.Fatalf("expected 7 pie slices (6 + Khác), got %d", len(labels))
	}
	if labels[6] != "Khác" || colors[6] != "#A1A1AA" {
		t.Fatalf("expected the 7th slice to be the reserved-gray Khác aggregate, got %q/%q", labels[6], colors[6])
	}
	if len(legend) != 7 || len(values) != 7 {
		t.Fatalf("expected 7 legend entries and values, got legend=%d values=%d", len(legend), len(values))
	}
}

func TestBuildBarSeriesZeroPadsMissingMonths(t *testing.T) {
	current := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	series := []sqlcgen.MonthlyTotalsSeriesRow{
		{Month: pgtype.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Valid: true}, TotalExpense: 5000, TotalIncome: 9000},
	}
	labels, chi, thu := buildBarSeries(series, current, 4)
	if len(labels) != 4 || len(chi) != 4 || len(thu) != 4 {
		t.Fatalf("expected 4 months of series data, got %d labels", len(labels))
	}
	if labels[3] != "Th 8" {
		t.Fatalf("expected the last label to be the current month (Th 8), got %q", labels[3])
	}
	if chi[3] != 5000 || thu[3] != 9000 {
		t.Fatalf("expected the current month's totals to carry through, got chi=%d thu=%d", chi[3], thu[3])
	}
	if chi[0] != 0 || thu[0] != 0 {
		t.Fatalf("expected a month with no data to zero-pad, got chi=%d thu=%d", chi[0], thu[0])
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/handlers/... -run TestComparisonText -run TestBuildPieData -run TestBuildBarSeries`
Expected: FAIL — `comparisonText`, `buildPieData`, `buildBarSeries` don't exist yet.

- [ ] **Step 3: Rewrite `internal/handlers/report_handlers.go`**

```go
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/sqlcgen"
)

const pieTopN = 6
const barMonths = 4

type pieLegendEntry struct {
	Name    string
	Color   string
	Percent string
	Amount  string
}

func dashboardPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		from, to := currentMonthRange()
		totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			http.Error(w, "could not load totals", http.StatusInternalServerError)
			return
		}

		prevFrom := pgDate(from.Time.AddDate(0, -1, 0))
		prevTotals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: prevFrom, OccurredOn_2: from})
		if err != nil {
			http.Error(w, "could not load previous month totals", http.StatusInternalServerError)
			return
		}
		hasPrevData := prevTotals.TotalExpense > 0 || prevTotals.TotalIncome > 0

		breakdown, err := deps.Queries.CategoryBreakdown(r.Context(), sqlcgen.CategoryBreakdownParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
		if err != nil {
			http.Error(w, "could not load breakdown", http.StatusInternalServerError)
			return
		}
		pieLabels, pieValues, pieColors, legend := buildPieData(breakdown, totals.TotalExpense)

		seriesFrom := pgDate(from.Time.AddDate(0, -(barMonths - 1), 0))
		series, err := deps.Queries.MonthlyTotalsSeries(r.Context(), sqlcgen.MonthlyTotalsSeriesParams{UserID: userID, OccurredOn: seriesFrom, OccurredOn_2: to})
		if err != nil {
			http.Error(w, "could not load monthly series", http.StatusInternalServerError)
			return
		}
		barLabels, barChi, barThu := buildBarSeries(series, from.Time, barMonths)
		hasAnyMonthData := false
		for _, v := range barChi {
			if v > 0 {
				hasAnyMonthData = true
			}
		}
		for _, v := range barThu {
			if v > 0 {
				hasAnyMonthData = true
			}
		}

		pieLabelsJSON, _ := json.Marshal(pieLabels)
		pieValuesJSON, _ := json.Marshal(pieValues)
		pieColorsJSON, _ := json.Marshal(pieColors)
		barLabelsJSON, _ := json.Marshal(barLabels)
		barChiJSON, _ := json.Marshal(barChi)
		barThuJSON, _ := json.Marshal(barThu)

		render(w, r, deps, "dashboard", "dashboard", map[string]any{
			"MonthLabel":        monthLabel(from.Time),
			"CurrentMonthValue": from.Time.Format("2006-01"),
			"TotalExpense":      totals.TotalExpense,
			"TotalIncome":       totals.TotalIncome,
			"ExpenseComparison": comparisonText(totals.TotalExpense, prevTotals.TotalExpense, hasPrevData),
			"IncomeComparison":  comparisonText(totals.TotalIncome, prevTotals.TotalIncome, hasPrevData),
			"CurrentMonthEmpty": totals.TotalExpense == 0 && totals.TotalIncome == 0,
			"HasAnyMonthData":   hasAnyMonthData,
			"PieLegend":         legend,
			"PieLabelsJSON":     template.JS(pieLabelsJSON),
			"PieValuesJSON":     template.JS(pieValuesJSON),
			"PieColorsJSON":     template.JS(pieColorsJSON),
			"BarLabelsJSON":     template.JS(barLabelsJSON),
			"BarChiJSON":        template.JS(barChiJSON),
			"BarThuJSON":        template.JS(barThuJSON),
		})
	}
}

// buildPieData turns CategoryBreakdown's already-total-desc-ordered rows
// into the top pieTopN slices plus a synthetic "Khác" aggregate for
// everything past that, per SPEC.md section 5.
func buildPieData(breakdown []sqlcgen.CategoryBreakdownRow, totalExpense int64) (labels []string, values []int64, colors []string, legend []pieLegendEntry) {
	shown := breakdown
	var otherSum int64
	if len(breakdown) > pieTopN {
		shown = breakdown[:pieTopN]
		for _, row := range breakdown[pieTopN:] {
			otherSum += row.Total
		}
	}
	for _, row := range shown {
		labels = append(labels, row.CategoryName)
		values = append(values, row.Total)
		colors = append(colors, row.CategoryColor)
		legend = append(legend, pieLegendEntry{
			Name: row.CategoryName, Color: row.CategoryColor,
			Percent: percentOf(row.Total, totalExpense), Amount: vnd(row.Total),
		})
	}
	if otherSum > 0 {
		labels = append(labels, "Khác")
		values = append(values, otherSum)
		colors = append(colors, "#A1A1AA")
		legend = append(legend, pieLegendEntry{
			Name: "Khác", Color: "#A1A1AA",
			Percent: percentOf(otherSum, totalExpense), Amount: vnd(otherSum),
		})
	}
	return
}

func percentOf(part, total int64) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%d%%", int(float64(part)/float64(total)*100+0.5))
}

// buildBarSeries returns exactly `months` consecutive [oldest..newest]
// points ending at currentMonthStart, zero-padding any month
// MonthlyTotalsSeries didn't return a row for.
func buildBarSeries(series []sqlcgen.MonthlyTotalsSeriesRow, currentMonthStart time.Time, months int) (labels []string, chi []int64, thu []int64) {
	byMonth := make(map[string]sqlcgen.MonthlyTotalsSeriesRow, len(series))
	for _, row := range series {
		byMonth[row.Month.Time.Format("2006-01")] = row
	}
	for i := months - 1; i >= 0; i-- {
		m := currentMonthStart.AddDate(0, -i, 0)
		key := m.Format("2006-01")
		labels = append(labels, shortMonthLabel(m))
		if row, ok := byMonth[key]; ok {
			chi = append(chi, row.TotalExpense)
			thu = append(thu, row.TotalIncome)
		} else {
			chi = append(chi, 0)
			thu = append(thu, 0)
		}
	}
	return
}

func shortMonthLabel(t time.Time) string {
	return fmt.Sprintf("Th %d", int(t.Month()))
}

// comparisonText builds SPEC.md section 5's "Tháng trước X · tăng/giảm Y%"
// line, or its "no data" fallback when the previous month had zero
// transactions of any kind.
func comparisonText(current, previous int64, hasPrevData bool) string {
	if !hasPrevData {
		return "Chưa có dữ liệu tháng trước"
	}
	if previous == 0 {
		return fmt.Sprintf("Tháng trước %s", vnd(previous))
	}
	diff := current - previous
	pct := int(float64(diff) / float64(previous) * 100)
	if pct < 0 {
		pct = -pct
	}
	if diff == 0 {
		return fmt.Sprintf("Tháng trước %s · không đổi", vnd(previous))
	}
	direction := "tăng"
	if diff < 0 {
		direction = "giảm"
	}
	return fmt.Sprintf("Tháng trước %s · %s %d%%", vnd(previous), direction, pct)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/handlers/... -run TestComparisonText -run TestBuildPieData -run TestBuildBarSeries`
Expected: PASS.

- [ ] **Step 5: Run the full test suite**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS (the two pre-existing dashboard tests still pass, per the design notes above — `dashboard.html` hasn't changed yet).

---

### Task 11: Tổng quan — template rewrite (stat cards, pie chart, bar chart, empty states, mobile)

**Files:**
- Rewrite: `internal/web/templates/dashboard.html`
- Modify: `internal/handlers/report_handlers.go` (add `?thang=` support to `dashboardPage`, matching the pattern Task 7 established for `/transactions`)
- Modify: `internal/handlers/report_handlers_test.go`
- Modify: `internal/handlers/smoke_test.go`

**Interfaces:**
- `dashboardPage` now delegates to `buildDashboardData(r, deps, userID, monthParam) (map[string]any, error)`, and branches on `HX-Request` the same way `transactionsPage` (Task 7) and `loginPage`/`registerPage` (Task 2) already do — full page normally, just the `dashboard_month_section` fragment when the month dropdown triggers it. `monthRangeFor` (Task 7, same package) is reused as-is.

Design notes:
- **The month dropdown wasn't in Task 10's scope** — SPEC.md section 5 puts one on the Tổng quan screen too ("nút chọn tháng (giống trang Giao dịch)"), but wiring it needs the same `HX-Request`-branching/`transactions_month_section`-style fragment this task is already adding the template for, so it's added here alongside the chart markup rather than as a separate task.
- **The Chart.js library `<script src>` tag lives in the outer, non-swapped `content` block; the data island and chart-init script live inside `dashboard_month_section`.** This matters for correctness, not just tidiness: htmx re-executes `<script>` tags found in swapped-in fragments (it extracts and re-runs them, unlike a plain `innerHTML` assignment), so the chart-init script re-running on every month switch is exactly what destroys the old `Chart` instances and builds new ones against the new data — but the *library* script tag must NOT be inside the swapped region, since re-fetching/re-declaring the global `Chart` constructor on every swap would be wasteful and could race with in-flight chart construction.
- **Pie chart mobile view shows the same top-6 legend as desktop, not top-4** — SPEC.md section 5's mobile variant says "Chỉ hiện 4 danh mục lớn nhất, còn lại gộp Khác", but the top-N cut is computed once, server-side, in Task 10's `buildPieData`, with no way to know the client's viewport at render time. Making it viewport-aware would mean either a second query variant or a client-side re-slice of already-rendered legend rows; given this is a minor content-count difference (top-6 vs top-4), not a token/color/spacing difference, this task ships top-6 everywhere and flags it as an open item for Task 12's visual QA rather than adding either mechanism preemptively.
- **The empty-state reconciliation from Task 10** (`CurrentMonthEmpty` / `HasAnyMonthData`) is implemented as: a full combined empty-state card only when *both* are true (no current-month data and no data anywhere in the 4-month window); otherwise the pie chart's own slot shows a smaller "Chưa có chi tiêu tháng này" message while the bar chart still renders normally.

- [ ] **Step 1: Fix the two pre-existing dashboard tests' now-invalid raw-number assertions**

In `internal/handlers/report_handlers_test.go`'s `TestDashboardShowsMonthlyTotal`, change:

```go
if !strings.Contains(dashRec.Body.String(), "100000") {
```

to:

```go
if !strings.Contains(dashRec.Body.String(), "100.000") {
```

In `internal/handlers/smoke_test.go`'s `TestEndToEndRegisterAddTransactionSeeDashboard`, change:

```go
if !strings.Contains(dashRec.Body.String(), "25000") {
```

to:

```go
if !strings.Contains(dashRec.Body.String(), "25.000") {
```

(Both now check for the `vnd`-formatted thousands-separated substring instead of the raw integer, which this task's template rewrite replaces.)

- [ ] **Step 2: Add new tests to `internal/handlers/report_handlers_test.go`**

Add `"expensetracker/internal/sqlcgen"` to the imports.

```go
func TestDashboardShowsPreviousMonthComparison(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "dash-compare@example.com", "s3cret-pass")
	ctx := context.Background()

	user, err := deps.Queries.GetUserByEmail(ctx, "dash-compare@example.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	categories, err := deps.Queries.ListCategoriesForUser(ctx, pgtype.Int8{})
	if err != nil || len(categories) == 0 {
		t.Fatalf("list categories: %v", err)
	}
	category := firstCategoryOfType(t, categories, "expense")
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	now := time.Now()
	if _, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 100000, Type: "expense",
		Description: "current", OccurredOn: pgtype.Date{Time: now, Valid: true},
	}); err != nil {
		t.Fatalf("create current-month transaction: %v", err)
	}

	prevMonth := now.AddDate(0, -1, 0)
	if _, err := deps.Queries.CreateTransaction(ctx, sqlcgen.CreateTransactionParams{
		UserID: user.ID, CategoryID: category.ID, Amount: 200000, Type: "expense",
		Description: "previous",
		OccurredOn:  pgtype.Date{Time: time.Date(prevMonth.Year(), prevMonth.Month(), 10, 0, 0, 0, 0, time.UTC), Valid: true},
	}); err != nil {
		t.Fatalf("create previous-month transaction: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Tháng trước") {
		t.Fatalf("expected a previous-month comparison line, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "giảm") {
		t.Fatalf("expected 'giảm' (current 100.000 < previous 200.000), got: %s", rec.Body.String())
	}
}

func TestDashboardEmptyStateWhenNoTransactions(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "dash-empty@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Chưa đủ dữ liệu để vẽ biểu đồ") {
		t.Fatalf("expected the empty-state message for a brand-new user with no transactions, got: %s", rec.Body.String())
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestDashboard`
Expected: FAIL — the old `dashboard.html` doesn't render comparison text or the empty-state copy yet.

- [ ] **Step 4: Extend `dashboardPage` in `internal/handlers/report_handlers.go` with month-param support**

Replace the `dashboardPage` function with:

```go
func dashboardPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := auth.UserIDFromContext(r.Context())

		data, err := buildDashboardData(r, deps, userID, r.URL.Query().Get("thang"))
		if err != nil {
			http.Error(w, "could not load dashboard", http.StatusInternalServerError)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			renderNamed(w, r, deps, "dashboard", "dashboard_month_section", "", data)
			return
		}
		render(w, r, deps, "dashboard", "dashboard", data)
	}
}

func buildDashboardData(r *http.Request, deps Deps, userID int64, monthParam string) (map[string]any, error) {
	from, to := monthRangeFor(monthParam)

	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}

	prevFrom := pgDate(from.Time.AddDate(0, -1, 0))
	prevTotals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: prevFrom, OccurredOn_2: from})
	if err != nil {
		return nil, err
	}
	hasPrevData := prevTotals.TotalExpense > 0 || prevTotals.TotalIncome > 0

	breakdown, err := deps.Queries.CategoryBreakdown(r.Context(), sqlcgen.CategoryBreakdownParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}
	pieLabels, pieValues, pieColors, legend := buildPieData(breakdown, totals.TotalExpense)

	seriesFrom := pgDate(from.Time.AddDate(0, -(barMonths - 1), 0))
	series, err := deps.Queries.MonthlyTotalsSeries(r.Context(), sqlcgen.MonthlyTotalsSeriesParams{UserID: userID, OccurredOn: seriesFrom, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}
	barLabels, barChi, barThu := buildBarSeries(series, from.Time, barMonths)
	hasAnyMonthData := false
	for i := range barChi {
		if barChi[i] > 0 || barThu[i] > 0 {
			hasAnyMonthData = true
		}
	}

	months, err := deps.Queries.ListDistinctTransactionMonths(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	currentFrom, _ := currentMonthRange()
	var available []map[string]any
	for _, m := range months {
		if m.Time.Year() == currentFrom.Time.Year() && m.Time.Month() == currentFrom.Time.Month() {
			continue
		}
		available = append(available, map[string]any{"Value": m.Time.Format("2006-01"), "Label": monthLabel(m.Time)})
	}

	pieLabelsJSON, _ := json.Marshal(pieLabels)
	pieValuesJSON, _ := json.Marshal(pieValues)
	pieColorsJSON, _ := json.Marshal(pieColors)
	barLabelsJSON, _ := json.Marshal(barLabels)
	barChiJSON, _ := json.Marshal(barChi)
	barThuJSON, _ := json.Marshal(barThu)

	return map[string]any{
		"MonthLabel":        monthLabel(from.Time),
		"CurrentMonthValue": currentFrom.Time.Format("2006-01"),
		"AvailableMonths":   available,
		"TotalExpense":      totals.TotalExpense,
		"TotalIncome":       totals.TotalIncome,
		"ExpenseComparison": comparisonText(totals.TotalExpense, prevTotals.TotalExpense, hasPrevData),
		"IncomeComparison":  comparisonText(totals.TotalIncome, prevTotals.TotalIncome, hasPrevData),
		"CurrentMonthEmpty": totals.TotalExpense == 0 && totals.TotalIncome == 0,
		"HasAnyMonthData":   hasAnyMonthData,
		"PieLegend":         legend,
		"PieLabelsJSON":     template.JS(pieLabelsJSON),
		"PieValuesJSON":     template.JS(pieValuesJSON),
		"PieColorsJSON":     template.JS(pieColorsJSON),
		"BarLabelsJSON":     template.JS(barLabelsJSON),
		"BarChiJSON":        template.JS(barChiJSON),
		"BarThuJSON":        template.JS(barThuJSON),
	}, nil
}
```

(Delete the old inline body of `dashboardPage` entirely — it's now split between the thin handler and `buildDashboardData`. `buildPieData`, `buildBarSeries`, `comparisonText`, `pieTopN`, `barMonths`, `pieLegendEntry` from Task 10 are unchanged.)

- [ ] **Step 5: Rewrite `internal/web/templates/dashboard.html`**

```html
{{define "content"}}
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4"></script>
<div class="max-w-[880px] mx-auto px-5 md:px-6 pt-4 md:pt-7 pb-24 md:pb-9 flex flex-col gap-[14px] md:gap-[18px]">
  {{template "dashboard_month_section" .}}
</div>
{{end}}

{{define "dashboard_month_section"}}
<div id="dashboard-month-section" class="flex flex-col gap-[14px] md:gap-[18px]">
  <div class="flex items-center justify-between">
    <h1 class="text-[19px] md:text-[18px] font-semibold text-ink">{{.MonthLabel}}</h1>
    <div class="relative">
      <button type="button" onclick="this.nextElementSibling.classList.toggle('hidden')" class="h-8 px-3 rounded-lg border border-border-input bg-white text-[13px] font-medium text-ink flex items-center gap-1">{{.MonthLabel}} <span class="text-placeholder">▾</span></button>
      <div class="hidden absolute right-0 top-9 z-20 bg-white border border-border-card rounded-lg py-1 w-[160px] shadow-sm">
        <button type="button" hx-get="/dashboard?thang={{.CurrentMonthValue}}" hx-target="#dashboard-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">Tháng này</button>
        {{range .AvailableMonths}}
        <button type="button" hx-get="/dashboard?thang={{.Value}}" hx-target="#dashboard-month-section" hx-swap="outerHTML" hx-push-url="true" class="block w-full text-left px-3 py-2 text-[13px] hover:bg-track">{{.Label}}</button>
        {{end}}
      </div>
    </div>
  </div>

  <div class="flex flex-col md:flex-row gap-3 md:gap-4">
    <div class="flex-1 bg-surface border border-border-card rounded-xl p-5 flex flex-col gap-2">
      <p class="text-[12px] font-medium text-ink-faint">Tổng chi tháng này</p>
      <p class="font-mono text-[27px] md:text-[32px] font-semibold text-expense" style="letter-spacing:-0.02em">{{if .CurrentMonthEmpty}}<span class="text-ink-zero">0₫</span>{{else}}{{vnd .TotalExpense}}{{end}}</p>
      {{if not .CurrentMonthEmpty}}<p class="text-[12px] text-ink-faint">{{.ExpenseComparison}}</p>{{end}}
    </div>
    <div class="flex-1 bg-surface border border-border-card rounded-xl p-5 flex flex-col gap-2">
      <p class="text-[12px] font-medium text-ink-faint">Tổng thu tháng này</p>
      <p class="font-mono text-[27px] md:text-[32px] font-semibold text-income" style="letter-spacing:-0.02em">{{if .CurrentMonthEmpty}}<span class="text-ink-zero">0₫</span>{{else}}{{vnd .TotalIncome}}{{end}}</p>
      {{if not .CurrentMonthEmpty}}<p class="text-[12px] text-ink-faint">{{.IncomeComparison}}</p>{{end}}
    </div>
  </div>

  {{if and .CurrentMonthEmpty (not .HasAnyMonthData)}}
  <div class="bg-surface border border-border-card rounded-xl text-center py-11 px-7">
    <p class="text-[15px] font-semibold text-ink">Chưa đủ dữ liệu để vẽ biểu đồ</p>
    <p class="text-[13px] text-ink-faint mt-1">Hai thẻ số liệu vẫn hiển thị 0₫. Biểu đồ xuất hiện sau giao dịch đầu tiên.</p>
    <a href="/transactions" class="inline-block mt-3 text-[13px] text-accent font-medium">Thêm giao dịch</a>
  </div>
  {{else}}
  <div class="flex flex-col md:flex-row gap-4">
    <div class="md:w-[440px] bg-surface border border-border-card rounded-xl p-[18px]">
      <p class="text-[13px] font-semibold text-ink mb-3">Chi theo danh mục</p>
      {{if .CurrentMonthEmpty}}
      <p class="text-[13px] text-ink-faint text-center py-8">Chưa có chi tiêu tháng này</p>
      {{else}}
      <div class="flex items-center gap-4">
        <canvas id="pie-chart" width="150" height="150" class="w-[108px] h-[108px] md:w-[150px] md:h-[150px] shrink-0"></canvas>
        <div class="flex-1 flex flex-col gap-[9px] min-w-0">
          {{range .PieLegend}}
          <div class="flex items-center gap-2 text-[12px]">
            <span class="w-2 h-2 rounded-full shrink-0" style="background-color: {{.Color}}"></span>
            <span class="flex-1 min-w-0 truncate text-[#57534E]">{{.Name}}</span>
            <span class="font-mono text-ink-faintest w-[34px] text-right shrink-0">{{.Percent}}</span>
            <span class="hidden md:inline font-mono font-medium w-[86px] text-right shrink-0">{{.Amount}}</span>
          </div>
          {{end}}
        </div>
      </div>
      {{end}}
    </div>
    <div class="flex-1 bg-surface border border-border-card rounded-xl p-[18px]">
      <div class="flex items-center justify-between mb-3">
        <p class="text-[13px] font-semibold text-ink">4 tháng gần nhất</p>
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1 text-[11px] text-ink-faint"><span class="w-2 h-2 rounded-[2px]" style="background-color:#C2410C"></span>Chi</span>
          <span class="flex items-center gap-1 text-[11px] text-ink-faint"><span class="w-2 h-2 rounded-[2px]" style="background-color:#2F7D5B"></span>Thu</span>
        </div>
      </div>
      <canvas id="bar-chart" class="w-full h-[118px] md:h-[158px]"></canvas>
    </div>
  </div>
  {{end}}

  <script type="application/json" id="chart-data">
    {"pie": {"labels": {{.PieLabelsJSON}}, "values": {{.PieValuesJSON}}, "colors": {{.PieColorsJSON}}}, "bars": {"labels": {{.BarLabelsJSON}}, "chi": {{.BarChiJSON}}, "thu": {{.BarThuJSON}}}}
  </script>
  <script>
    (function () {
      if (window.__pieChart) { window.__pieChart.destroy(); window.__pieChart = null; }
      if (window.__barChart) { window.__barChart.destroy(); window.__barChart = null; }
      var dataEl = document.getElementById('chart-data');
      if (!dataEl) return;
      var chartData = JSON.parse(dataEl.textContent);

      var pieCanvas = document.getElementById('pie-chart');
      if (pieCanvas && chartData.pie.labels.length > 0) {
        window.__pieChart = new Chart(pieCanvas, {
          type: 'doughnut',
          data: {
            labels: chartData.pie.labels,
            datasets: [{ data: chartData.pie.values, backgroundColor: chartData.pie.colors, borderWidth: 2, borderColor: '#fff' }]
          },
          options: { cutout: '62%', plugins: { legend: { display: false }, tooltip: { enabled: false } }, animation: false }
        });
      }

      var barCanvas = document.getElementById('bar-chart');
      if (barCanvas) {
        window.__barChart = new Chart(barCanvas, {
          type: 'bar',
          data: {
            labels: chartData.bars.labels,
            datasets: [
              { label: 'Chi', data: chartData.bars.chi, backgroundColor: '#C2410C', borderRadius: 3, barPercentage: 0.62, categoryPercentage: 0.6 },
              { label: 'Thu', data: chartData.bars.thu, backgroundColor: '#2F7D5B', borderRadius: 3, barPercentage: 0.62, categoryPercentage: 0.6 }
            ]
          },
          options: {
            animation: false,
            plugins: { legend: { display: false } },
            scales: {
              x: { grid: { display: false }, border: { display: false }, ticks: { color: '#9C9891', font: { family: 'JetBrains Mono' } } },
              y: {
                beginAtZero: true, grid: { color: '#F1EFEC' }, border: { display: false },
                ticks: {
                  color: '#B5B1AA', maxTicksLimit: 4, font: { family: 'JetBrains Mono' },
                  callback: function (value) { return value >= 1000000 ? (value / 1000000) + 'tr' : value; }
                }
              }
            }
          }
        });
      }
    })();
  </script>
</div>
{{end}}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `TEST_DATABASE_URL=<dsn> go test ./internal/handlers/... -run TestDashboard -run TestEndToEnd`
Expected: PASS.

- [ ] **Step 7: Run the full test suite**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS.

- [ ] **Step 8: Manual smoke check**

Run: `go run ./cmd/server`, visit `/dashboard`: confirm both stat cards, the comparison line (add a transaction, check it updates), the pie chart + legend, and the bar chart all render; switch the month dropdown and confirm the charts destroy/redraw without a full page reload (check the Network tab shows no full-document request, and there's no console error about a canvas already in use); resize to ~390px and confirm the pie chart's amount column disappears and the bar/pie cards stack vertically.

---

### Task 12: Responsive/polish — Playwright visual QA pass at 390px and 1280px

**Files:** none pre-determined — this task finds and fixes real rendering issues against SPEC.md, so its file list is whatever Step 3 turns up. Every fix must be a real edit (no placeholder "TODO: revisit" comments) before this task is considered done.

Unlike Tasks 1–11, this is a verification-and-fix task, not a build-from-a-known-diff task, so its steps are a concrete checklist to execute with the `mcp__plugin_playwright_playwright__*` tools (already available in this environment) rather than a code diff to apply.

- [ ] **Step 1: Seed data covering every state the checklist needs**

Start Postgres and the app (`go run ./cmd/server`), then, via the running app's own UI or direct SQL, get to a state with:
- One brand-new account with **zero** categories beyond the 9 defaults and **zero** transactions (for empty states on Giao dịch/Danh mục/Tổng quan).
- A second account with: at least 2 custom categories (one expense, one income), at least 9 expense transactions across at least 3 different expense categories in the current month (enough to trigger the pie chart's top-6-plus-Khác capping), at least 1 income transaction this month, and at least 1 transaction in each of 2 prior months (to exercise the month dropdown and the bar chart's zero-padding on any month left empty).

- [ ] **Step 2: Open the browser and iterate every (screen, breakpoint) pair**

For each of the 5 screens — Đăng nhập/Đăng ký, Giao dịch, Danh mục, Tổng quan, and the nav (checked as part of every other screen's snapshot) — and each breakpoint — 390px and 1280px width (`browser_resize`) — use `browser_navigate` + `browser_snapshot` (and `browser_take_screenshot` for anything a snapshot's accessibility tree can't judge, like exact spacing/color) to inspect the rendered page, logged in as the seeded populated account first, then again on the empty account for the screens that have an empty state (Giao dịch, Danh mục, Tổng quan).

- [ ] **Step 3: Check each screen against this SPEC.md-derived list, fixing any real deviation found**

**Đăng nhập/Đăng ký** (§2): 380px centered card at 1280px, no card chrome at 390px (title left-aligned, logo left-aligned); tab switch swaps only the card body (Network tab shows no full-document request); wrong password shows the red error block with the exact copy `Email hoặc mật khẩu không đúng.`; register→login tab switch preserves nothing stale from the other tab.

**Giao dịch** (§3): quick-add form field widths approximate the spec table at 1280px; changing Loại reloads the danh mục select/chips; adding a transaction prepends it and updates Chi/Thu/count/"Còn lại" without a reload; month dropdown lists months with data + "Tháng này", updates the URL (`?thang=`); Sửa swaps a row to inline inputs; Xóa shows the pink inline confirm bar (desktop) or the action sheet (mobile) — never a browser `confirm()`; empty state shows the exact copy from §3.3; at 390px: two stat cards, FAB positioned near the bottom-right, bottom sheet opens with the 56px amount field and chip-based category picker.

**Danh mục** (§4): two groups (Chi/Thu) each with a heading + card; default categories show only "Đổi màu" (no Xóa button in the DOM, not just visually hidden — check via snapshot's accessibility tree); color popover offers exactly the 8 swatches, never the reserved gray; deleting a custom expense category with transactions shows the reassign-to-"Khác" copy; deleting a custom income category with transactions shows only a dismiss action; empty-state nudge appears when there are no custom categories; at 390px: `+` button opens the bottom sheet with the same form.

**Tổng quan** (§5): two stat cards with the comparison line (`Tháng trước ... · tăng/giảm N%`); pie chart + legend sorted descending, capped with a `Khác` entry when there are more than 6 expense categories; bar chart shows exactly 4 months with a Chi/Thu legend; switching the month dropdown redraws both charts (check `browser_console_messages` for a "Canvas is already in use" error — its absence confirms the destroy-before-recreate logic works); empty state (fresh account) shows `Chưa đủ dữ liệu để vẽ biểu đồ` while the two stat cards still show `0₫`; at 390px: stat cards stack vertically, pie legend drops the amount column, both chart cards stack.

**Nav** (§6): desktop top bar sticky at `top:0`, active link styled with the accent background; mobile bottom bar fixed, active dot accent-colored; logout opens a confirm dialog on mobile, submits directly (still via the hidden-field CSRF form) on desktop; page content isn't clipped by the fixed mobile bottom bar (check for a `pb-24`-equivalent gap under the last visible content).

Fix each real deviation as a normal code edit in the relevant template/handler file (adding it to the Files list above as you go); re-run `browser_navigate` + `browser_snapshot`/`browser_take_screenshot` on the specific screen/breakpoint after each fix to confirm it.

- [ ] **Step 4: Check `browser_console_messages` on every screen for JS errors**

Particularly: no Tailwind CDN warnings escalated to errors, no `htmx:configRequest`/CSRF header issues (a 403 on any mutating request means the meta tag or the `hx-on:transaction-created` wiring broke), no Chart.js canvas-reuse errors.

- [ ] **Step 5: Run the full automated test suite one final time**

Run: `TEST_DATABASE_URL=<dsn> go test ./...`
Expected: PASS. If any Step 3 fix touched Go code, this is the final confirmation nothing regressed.

- [ ] **Step 6: Close out**

Summarize what was found and fixed (or confirm nothing needed fixing) for the human — this is the last task in the plan.

---

## Self-Review

- **Spec coverage**: every numbered SPEC.md section (§1 tokens, §2 auth, §3 giao dịch incl. mobile, §4 danh mục incl. mobile, §5 tổng quan incl. mobile, §6 nav, §7 other states, §8 data constraints, §9 assets) is implemented by at least one task above — §1 in Task 1 (Tailwind config) and scattered through every template; §2 in Task 2; §3 in Tasks 5/6/7/8/9; §4 in Tasks 3/4; §5 in Tasks 10/11; §6 in Task 1; §7 (loading/error/success states) is covered implicitly by htmx's default button-disable-on-request behavior plus the per-form inline error patterns established in Tasks 2/4/6 — no dedicated task was needed since SPEC.md section 7 doesn't ask for anything beyond what those patterns already produce; §8 in Task 5 (transaction validation) and Task 4 (category name uniqueness); §9 (fonts, no icon library) in Task 1.
- **Placeholder scan**: every task's code blocks are complete, compilable-as-written Go/HTML/SQL — no `// TODO`, no `...`, no "implement this" prose standing in for code. The two deliberately-scoped exceptions are documented, not placeholders: Task 12 has no pre-written file list (it's a find-and-fix task by nature) and its checklist items are concrete rather than code, and several tasks note a documented simplification (mobile long-press → tap, generic vs per-field quick-add errors, top-6-everywhere pie legend) rather than leaving a TODO for later.
- **Type/name consistency across tasks**: `Deps` (auth_handlers.go) is referenced identically everywhere; `render`/`renderNamed` signatures introduced in Task 1 are used with that exact signature through Task 12; `categoryRowData`/`firstCategoryOfType`/`csrfTokenFor`/`withCSRF` helpers are each defined once and reused by every later task that needs them; sqlc-generated type names (`ListCategoriesWithTransactionCountsRow`, `MonthlyTotalsSeriesRow`, etc.) match the field names their consuming templates/handlers reference.

---
