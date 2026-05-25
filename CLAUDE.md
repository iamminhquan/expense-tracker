# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

## 5. Go Conventions

**Áp dụng cho `back-end/expense-tracker/`.**

**Naming**
- Exported `PascalCase`, unexported `camelCase`. Không `snake_case`.
- Package name: viết thường, ngắn, không underscore, không số nhiều (`expense`, không `expenses`).
- Tránh stutter: `expense.New()`, không `expense.NewExpense()`.
- Receiver: 1–2 ký tự, nhất quán trong cùng type (`func (s *UserService)` thì xuyên suốt dùng `s`).

**Error handling**
- Luôn check error. Không `_ = someCall()` trừ khi có comment lý do.
- Wrap khi truyền lên: `fmt.Errorf("create expense: %w", err)`.
- Sentinel errors → `errors.Is`. Typed errors → `errors.As`.
- Không leak lỗi GORM lên handler. Repository nuốt `gorm.ErrRecordNotFound`, trả lỗi domain (`ErrExpenseNotFound`) cho service.

**Layering (project-specific)**
- Handler: parse request, gọi service, format response. Không business logic.
- Service: không biết `*gin.Context`. Nhận/trả primitive hoặc struct, không nhận `*http.Request`.
- Repository: trả model, không trả `*gorm.DB`.
- Model: không import handler / service / repository.

**Khác**
- `context.Context` là param đầu tiên cho function block-được hoặc cần cancel.
- Interface nhỏ, định nghĩa ở phía consumer (service), không ở phía producer (repository).
- Không `panic` ngoài `main.go` / `init()`.
- Test: table-driven với `t.Run(name, ...)`. File `*_test.go` cùng package.
- `go fmt ./...` và `go mod tidy` trước commit.

## 6. TypeScript / React / Next.js Conventions

**Áp dụng cho `front-end/expense-tracker/` (Next.js 16, React 19, App Router).**

**TypeScript**
- `strict: true` luôn bật. Không tắt rule để cho qua.
- Không `any`. Dùng `unknown` rồi narrow. `// @ts-expect-error` phải kèm lý do.
- Explicit return type cho hàm exported. Hàm nội bộ infer là ổn.
- `type` cho union/intersection/utility. `interface` cho object shape có thể extend.
- Tránh `!` (non-null assertion) — narrow tử tế thay vào đó.
- `const` mặc định, `let` khi cần reassign, không `var`.
- `readonly` cho prop và field không mutate.

**React / Next.js**
- Server Component mặc định. Chỉ `"use client"` khi cần state, effect, browser API, hoặc event handler.
- File: kebab-case (`dashboard-shell.tsx`). Component: PascalCase. Hook: `useXxx`.
- Props interface tên `XxxProps`, khai báo cạnh component.
- Không động `localStorage` / `window` trong code chạy SSR. Bọc bằng `useIsClient()` hoặc `useEffect`.
- Co-locate: component / hook / util chỉ dùng trong 1 feature thì để cùng folder feature.
- Effect tối thiểu. Derive được từ props/state thì không cần effect.
- `key` trong list phải stable, không dùng index trừ khi list immutable.

**Styling (Tailwind)**
- Thứ tự class: layout → spacing → sizing → typography → color → state (`hover:`, `focus:`).
- Pattern lặp 3+ lần → extract component, không `@apply`.
- Dùng design token từ `globals.css` / shadcn, không hardcode hex trong JSX.

**Data fetching**
- Mọi call backend qua `lib/expenses.ts` hoặc lib tương đương. Component không `fetch` trực tiếp.
- Token truyền qua param, lib không tự đọc `localStorage`.

## 7. Commit Conventions

**Conventional Commits, atomic, KHÔNG gộp.**

**Format**
```
<type>(<scope>): <subject>

<body, optional>

<footer, optional>
```

Blank line giữa subject và body **bắt buộc** — thiếu thì git tool đọc sai.

**Types:** `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `style`, `perf`, `build`, `ci`.

**Scope**
- Top-level: `front-end` hoặc `back-end`, **viết đầy đủ — không `fe` / `be`**.
- Sub-scope: tên module / component / layer cụ thể, ngăn cách bằng `/`. Ví dụ: `front-end/expense-editor`, `back-end/handlers`, `back-end/repository/expense`.
- Cross-cutting: scope đơn — `deps`, `ci`, `docs`, `config`.

**Subject**
- Imperative: "add", "fix", "remove" — không "added", "fixes".
- Viết thường, không dấu chấm cuối.
- **Hard limit 100 ký tự**, bao gồm cả `type(scope):`. Càng ngắn càng tốt.
- Subject sắp chạm 100 → **tách commit**, không nhồi thêm "and" / "with" / "also".

**Body** (optional)
- Giải thích *why*, không *what*. *What* đọc diff là thấy.
- Wrap mỗi dòng ~72 ký tự cho `git log` đọc được. URL / path dài để nguyên.
- **Nên có** khi: fix bug (giải thích nguyên nhân), có breaking change, có quyết định non-obvious cần biện minh, reference issue/PR.
- **Không cần** khi: thay đổi tự giải thích qua diff (rename biến, format, thêm field đơn giản).

**Atomic commits — bắt buộc tách nhỏ**

Một commit = một thay đổi logic. KHÔNG gom nhiều thay đổi vào một commit kể cả khi cùng phục vụ một feature.

Tách theo các trục:
- **Theo layer:** model → repository → service → handler → route là 5 commit riêng.
- **Theo domain:** backend và frontend KHÔNG BAO GIỜ chung một commit.
- **Theo intent:** refactor và feature tách. Format/rename và logic change tách. Thêm dep và dùng dep tách.
- **Theo file phụ:** rename file commit riêng. Move file commit riêng trước khi sửa nội dung.

Test: nếu subject cần "and" / "with" / "also" để mô tả → commit đang làm quá nhiều, tách ra.

**Ví dụ — thêm field `note` cho Expense:**

```
feat(back-end/models): add Note field to Expense
feat(back-end/repository): persist Note in Create and Update
feat(back-end/handlers): accept note in expense request DTO
feat(front-end/lib): include note in expense API payloads
feat(front-end/expense-editor): add note textarea to form
```

5 commit, không phải 1.

**Ví dụ commit có body:**

```
fix(back-end/repository): return ErrExpenseNotFound for missing rows

GORM's First() returns gorm.ErrRecordNotFound which leaks
ORM details to the service layer. Wrap it in a domain error
so handlers can map to 404 without importing gorm.

Closes #47
```

**Khi nào KHÔNG cần tách**
- Đổi tên 1 symbol và cập nhật caller — đây là 1 thay đổi.
- Sửa typo trong 1 file.
- Fix lỗi do commit ngay trước trong cùng PR — `--amend` nếu chưa push.

**Trước khi commit**
- Backend: `go fmt ./... && go test ./...`
- Frontend: `bun run lint && bun run build`
- Diff phải đọc được trong 60 giây. Nếu không, tách nhỏ nữa.

## Commands

**Backend** — run from `back-end/expense-tracker/`:
```
go run .          # start API on :8080 (or $APP_PORT)
go test ./...     # run all tests
go fmt ./...      # format before committing
go mod tidy       # clean up after dependency changes
```

**Frontend** — run from `front-end/expense-tracker/`:
```
bun install       # install from bun.lock
bun dev           # dev server on :3000
bun run build     # production build (also type-checks)
bun run lint      # ESLint
```

For frontend changes, always run `bun run lint` and `bun run build` before finishing. The build is the authoritative type check.

## Architecture

### Monorepo layout

Two independent application roots, each with their own module/package manager:
- `back-end/expense-tracker/` — Go 1.23, module `github.com/iamminhquan/expense-tracker`
- `front-end/expense-tracker/` — Next.js 16, React 19, managed with Bun

### Backend — request flow

```
Gin router (routes/routes.go)
  └─ middleware.CORS + middleware.AuthRequired (JWT → sets userID in context)
       └─ handlers/ (parse request, call service, write response)
            └─ services/ (business logic, owns auth-secret for JWT signing)
                 └─ repositories/ (GORM queries, return domain models)
                      └─ models/ (User, Expense structs)
```

`main.go` wires everything: loads `config/`, opens `database/` (GORM + AutoMigrate), calls `routes.Setup()`.

`internal/auth/` provides `GenerateToken` and `ParseToken` helpers used by `UserService` and `AuthRequired` middleware respectively.

### Current API routes

```
GET  /health
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/auth/me            ← bearer token required
GET  /api/v1/expenses           ← bearer token required
POST /api/v1/expenses
PUT  /api/v1/expenses/:id
DELETE /api/v1/expenses/:id
```

`/api/v1/users` is a legacy alias for `/api/v1/auth/register`.

### Frontend — data flow

`app/expenses/page.tsx` is the only protected route. On mount it reads `auth_token` from `localStorage` (via `lib/auth.ts`), calls `GET /auth/me` to validate, then renders `<DashboardShell token user onSignOut>`. Any failure redirects to `/login`.

`lib/expenses.ts` owns all expense CRUD calls. Every function takes a `token` string and sends `Authorization: Bearer <token>`.

### Dashboard component structure

`components/dashboard/dashboard-shell.tsx` is the orchestrator: it holds all React state (expenses list, form, editing ID, loading flags) and passes slices down as props. It never renders UI directly.

Supporting components (each a focused presentational unit):

| File | Responsibility |
|---|---|
| `sidebar.tsx` | Desktop nav, user profile, logout |
| `topbar.tsx` | Search bar, notifications, month total |
| `mobile-nav.tsx` | Fixed bottom nav on mobile |
| `balance-chart.tsx` | 6-month AreaChart (Recharts) with gradient header |
| `spending-overview.tsx` | Donut PieChart + category breakdown list |
| `monthly-snapshot.tsx` | Income / expenses / savings tiles |
| `recent-activity.tsx` | Transaction list with category icons and edit/delete |
| `expense-editor.tsx` | Add / edit form |
| `quick-actions.tsx` | Quick-action button grid |
| `savings-goals.tsx` | Goal progress bars |
| `utils.ts` | `summarizeExpenses`, `formatCurrency`, `formatDate`, `getCategoryStyle` |
| `hooks.ts` | `useIsClient` — SSR-safe guard for Recharts rendering |

Charts use `useIsClient()` to avoid SSR hydration mismatches. Pass `summary: ExpenseSummary` (returned by `summarizeExpenses`) as the shared data shape across all chart and stats components.

### Environment variables (backend)

```
APP_PORT            default 8080
DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE
AUTH_SECRET         signs JWT tokens (HS256, 24h expiry)
CORS_ALLOWED_ORIGIN default http://localhost:3000
```

Frontend reads `NEXT_PUBLIC_API_BASE_URL` (default `http://localhost:8080/api/v1`).