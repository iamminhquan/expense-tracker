# $pend Frontend Context

## Purpose

- This file is the browser-facing source-of-truth for agent work on $pend: the `client/` React SPA, everything it renders and every request it makes to `server/`'s JSON API.
- `client/` is a Vite + React + TypeScript + Tailwind v4 single-page app (a standalone pnpm package: its own `package.json` and `pnpm-lock.yaml`, no workspace). There is no server-rendered HTML anywhere in this repo — see `.claude/context/server.md` for the JSON API half of the same product.
- The four main pages are `/dashboard`, `/transactions` (plus `/transactions/import`), `/categories`, `/settings`, behind a shared authenticated `Layout`. The pre-auth routes are `/login`, `/register` (one shared `AuthPage` component, tabbed), `/forgot-password`, `/reset-password`, `/verify-email`.

## Read This First

- Treat this file as the frontend source-of-truth for agent work, alongside `.claude/context/server.md` (the API half) and `.claude/context/README.md` (the directory's own index).
- The detailed rules live in `.claude/rules/*.md`, each loaded automatically for the files its `paths:` frontmatter covers, so there is no list of them to keep here. Repeatable workflows (writing components, committing, and so on) live in `.claude/skills/`, each offered by its own `description:`.
- Prefer reading the actual components over this prose — including this file — when they disagree. `App.tsx`'s route table and `lib/api/types.ts` are the two files least likely to drift without something breaking at compile or runtime, so they're the most reliable sources.

## Maintenance Rule

- Any agent that changes frontend behavior MUST review and update this file (and the relevant `.claude/rules/*.md` file) before finishing work.
- Update this file when any of these change:
  - the page/route surface (`App.tsx`)
  - the API client contract (`lib/api/*.ts`, `lib/api/types.ts`) — especially if it drifts from what `server/internal/api` actually returns
  - auth/token handling (`lib/auth/AuthContext.tsx`, `lib/api/tokenStore.ts`, `lib/api/client.ts`)
  - the CSS variable palette or the theme mechanism
  - the TanStack Query cache-invalidation strategy
  - build tooling (`vite.config.ts`, `package.json` scripts)
- If a change does not affect behavior, no rewrite is required, but check whether the existing text is still accurate.
- Keep updates short and factual — this is not a changelog of component diffs; the Change Log section at the bottom is, one line per change.

## Stack

- Build: Vite, TypeScript, pnpm — a standalone package, with `client/pnpm-lock.yaml` as its lockfile and `packageManager` in `client/package.json` pinning the pnpm version (what CI and Vercel read). There is no pnpm workspace and no root `package.json`.
- UI: React 19, Tailwind v4 (`@tailwindcss/vite`, build-time, no CDN), `react-router-dom` for client-side routing.
- Data: `@tanstack/react-query` for every server-state fetch/mutation/cache; there is no Redux/Zustand/global-store layer beyond `AuthContext`/`ThemeContext` and TanStack Query's own cache.
- Charts: Chart.js via `react-chartjs-2`, Dashboard-only, lazy-loaded with the rest of that page's chunk.
- Icons: `lucide-react` (tree-shaken, imported one icon at a time).
- Fonts: Be Vietnam Pro (body) and Bricolage Grotesque (headings, every money figure, the wordmark), from Google Fonts in `index.html`. Both cover Vietnamese and ₫.
- Lint: `oxlint`.
- No CSS-in-JS and no third-party component library. Every component is hand-written Tailwind against the tokens in `index.css`, with the shared primitives in `components/ui/`.

## Main Scope Right Now

- The visual design is "Pocket Mono": a near-black accent on warm off-white (inverted in dark mode), indigo for expense and green for income, 28px-radius cards, and display-font money figures. The design handoff's token table is what `index.css` implements.
- `Layout.tsx` wraps every authenticated page with a desktop header (nav links, the balance widget, the user menu) and a mobile header plus a floating icon tab bar (both in the DOM, each hidden at the breakpoint the other owns via Tailwind's `md:` prefix). On mobile the balance widget moves into the user menu.
- Four authenticated pages (`DashboardPage`, `TransactionsPage`, `CategoriesPage`, `SettingsPage`) plus `ImportPage` (CSV) and the five pre-auth pages.
- The two pages with the most real complexity: `TransactionsPage` (filters, month scope including "all months", pagination, inline create/edit/delete, the duplicate-transaction badge, long-press mobile gesture) and `DashboardPage` (the doughnut + bar chart pair via Chart.js).
- Theming (`auto`/`light`/`dark`) and the mobile long-press/bottom-sheet gesture layer are both load-bearing UI, ported deliberately from the deleted HTML app rather than left as a gap — see `.claude/rules/theming.md` and `.claude/rules/mobile-nav.md`.

## What Is Actually Implemented

- `client/src/App.tsx`: the route table. Every page component is `lazy()`-loaded (one chunk per route, behind a single `<Suspense>` boundary) — Chart.js alone is ~170KB, and this keeps it out of a signed-out visitor's first paint.
- `client/src/lib/api/`: the whole API client.
  - `client.ts` — the one place that calls `fetch`. Unwraps the server's `{success, message, data}` envelope (`readApiResponse`, also used by `import.ts` and `AuthContext.tsx`) so hooks get `data` and errors become `ApiError(status, message)`. Attaches the access token, retries once through a silent refresh on a 401, collapses concurrent refreshes into one request.
  - `tokenStore.ts` — the access token itself, a plain module variable (never `localStorage`/`sessionStorage` — the migration's locked JWT-storage decision), so `client.ts` can read it without importing React/`AuthContext` and creating a cycle.
  - `types.ts` — hand-written TypeScript types mirroring `server/internal/api`'s DTOs (the migration's locked type-sync decision: no codegen).
  - One file per resource: `auth.ts`, `categories.ts`, `transactions.ts`, `dashboard.ts`, `settings.ts`, `import.ts` (the one multipart/file-upload exception, bypassing `client.ts`'s always-JSON `request()`).
- `client/src/lib/auth/AuthContext.tsx`: owns login/register/logout and the silent-refresh bootstrap every page load runs (the access token doesn't survive a reload by design; the refresh-token cookie does). `ProtectedRoute.tsx` gates every authenticated route on its `status`.
- `client/src/lib/theme/ThemeContext.tsx`: applies the signed-in user's theme (carried on `/api/v1/me`/`/api/v1/refresh`'s response) to `<html>`. Must be nested inside `AuthProvider`.
- `client/src/hooks/`: one TanStack Query hook module per resource (`useCategories`, `useTransactions`, `useDashboard`, `useSettings`) — queries plus mutations, with mutations invalidating whatever else their write affects (see Data Layer below).
- `client/src/components/layout/`: `Layout`, `ProtectedRoute`, `AuthLayout` (the pre-auth card shell), `BalanceWidget` (spent-ratio ring plus balance), `UserMenu` (Settings, the theme switch, log out; the balance too on mobile).
- `client/src/components/`: `MonthPicker` (a keyboard-driven listbox, not a native `<select>`), `BottomSheet` (drag-to-dismiss).
- `client/src/components/ui/`: the design system's primitives, used across pages: `Field` (label + control + hint/error, wired with `useId`), `FieldErrorText`, `SelectControl`, `AmountInput` (₫ suffix), `PasswordInput` (show/hide), `SegmentedControl`, `Checkbox`, `Badge`, `Banner`, `InlineError` (with Retry), `EmptyState`, `PageSkeleton`/`SkeletonBar`, `ConfirmDialog` (a centered `<dialog>` on desktop, a bottom sheet on mobile; replaces `confirm()`), `StatusIcon`.
- `client/src/lib/formStyles.ts`: the shared class strings: `buttonClass(variant, size)`, `iconButtonClass`, `inputClass`, `selectClass`, `labelClass`, `cardClass`, `cardTitleClass`, `pageTitleClass`.
- `client/src/lib/toast/ToastContext.tsx`: `useToast()` for success/error toasts after a mutation; the provider sits inside `ThemeProvider`.
- `client/src/lib/categorySwatches.ts`: the 8 selectable category colors (see `.claude/rules/theming.md`).
- `client/src/hooks/useMediaQuery.ts` (`useIsDesktop`), `hooks/useDismiss.ts` (Escape/outside-click for popovers), `hooks/useThemeColors.ts` (the active theme's chart colors, read from the CSS variables).
- `client/src/hooks/useLongPress.ts`: the mobile long-press gesture hook.
- `client/src/lib/format.ts`: client-side display formatting (money, dates, timestamps) — the backend ships raw numbers and leaves this to the client, see `.claude/context/server.md`'s Request and Response Conventions.
- `client/src/lib/charts.ts`: registers Chart.js's elements once and sets its default font, imported for its side effect.

## Important Reality Checks

- The access token lives in memory only (a module variable in `tokenStore.ts`) and does not survive a page reload by design. Every page load runs `AuthContext`'s silent-refresh bootstrap (a `POST /api/v1/refresh` using the httpOnly cookie) before rendering anything behind `ProtectedRoute`. Don't "fix" a reload losing auth state by persisting the access token to storage — that's the locked decision this migration made, not a bug.
- `TransactionsPage` and `DashboardPage` keep their filters/month in the URL's own query string (`useSearchParams`), never local `useState`. This isn't a style preference — a real browser test caught the bug that happens otherwise (a link to a specific month silently reset to the current one because nothing read the URL it landed on). See `.claude/rules/req-value-objects.md`.
- A mutation (create/update/delete a transaction or category) invalidates more than its own TanStack Query key: deleting a category can reassign transactions, a transaction changes totals the dashboard and a category's `transactionCount` both depend on. See each `hooks/use*.ts` file's invalidation calls rather than assuming one key is enough when adding a new mutation.
- Chart.js paints on a canvas and can't follow a CSS variable. `useThemeColors()` reads the palette from `index.css` and changes `key` on every theme switch; the charts are keyed on it, so they rebuild (without animating) in the new colors. Updating the existing chart in place with `updateMode="none"` left the bars in the old color, so don't go back to that. See `.claude/rules/dashboard.md`.
- Rows and dialogs that differ between phone and desktop choose their layout with `useIsDesktop()` and render one variant, rather than rendering both and hiding one with CSS, so a screen reader never meets duplicate controls. Only `Layout.tsx`'s two headers use the CSS approach (`.claude/rules/mobile-nav.md`).
- There is no client-side reminder for an unverified email address yet, and `userDTO`/`User` doesn't carry `EmailVerified`. See `.claude/rules/email-verification.md`'s known gap.

## Frontend Layout

- `client/src/App.tsx`: route table, lazy-loading, the provider tree (`QueryClientProvider` → `AuthProvider` → `ThemeProvider` → `ToastProvider`).
- `client/src/pages/`: one folder per page, holding the page plus one file per sub-component: `pages/dashboard/` (`KpiCards`, `SpendingDoughnut`, `MonthlyBars`), `pages/transactions/` (`FilterBar`, `AddTransactionForm`, `TransactionRow`, and `rowLayout.ts` for the grid the header and rows share), `pages/import/` (`ImportStepper`, `UploadStep`, `FileRow`, `MappingForm`, `PreviewPanel`), `pages/categories/` (`CategoryRow`, `AddCategoryForm`, `SwatchPicker`), `pages/settings/` (one file per card, plus the `Card` frame they share), and `pages/auth/` for the five pre-auth pages (`AuthPage` composes `AuthTabs`, `LoginForm`, `RegisterForm`).
- `client/src/components/`: shared components. `components/ui/` holds the design system's primitives; `components/layout/` holds the two app shells: the authenticated one (`Layout`, `ProtectedRoute` and the header widgets) and the pre-auth `AuthLayout`.
- `client/src/hooks/`: TanStack Query hooks, one module per resource.
- `client/src/lib/api/`: the API client — see What Is Actually Implemented above.
- `client/src/lib/auth/`, `client/src/lib/theme/`, `client/src/lib/toast/`: the three app-wide React contexts.
- `client/src/lib/format.ts`, `client/src/lib/formStyles.ts`, `client/src/lib/categorySwatches.ts`, `client/src/lib/charts.ts`, `client/src/lib/queryClient.ts`: small standalone utilities.

## Data Layer (TanStack Query)

- Every server read is a `useQuery`; every write is a `useMutation` whose `onSuccess` calls `queryClient.invalidateQueries` for every query key the write could have changed — the client-side replacement for the old HTML app's hand-wired out-of-band swaps (`header_balance_oob`, `totals_oob`).
- `useTransactions.ts`'s `invalidateEverythingATransactionTouches` is the pattern to copy: a transaction write invalidates `['transactions']`, `['dashboard']`, and `['categories']` together, because a delete can reassign a category's transactions to "Other" and any write changes totals the dashboard shows.
- `Layout.tsx` calls `useDashboard()` once at the top of the authenticated route tree specifically so every page shares the one cached `headerBalance` request rather than each page issuing its own.

## Theming

See `.claude/rules/theming.md` for the full rules (CSS variable palette, light/dark/auto resolution, the category swatch list). Summary: all color flows through `index.css`'s `@theme`-mapped CSS variables (semantic names: `app`, `surface`, `surface-2`, `border`, `border-strong`, `ink`, `ink-muted`, `accent`, `on-accent`, `expense`, `income`, `chart-income`, `danger`, `warning`, each `*-tint`, `scrim`), the dark palette is declared twice (media query + explicit class) so no JS is needed at load time, and `ThemeContext.tsx` is what sets the class once a user's preference is known. `index.css` also holds the motion tokens (`--dur-*`, `--ease-*`) and the `animate-*` keyframes; under `prefers-reduced-motion` the transforms drop out but fades and colour changes stay.

## Mobile Navigation and Gestures

See `.claude/rules/mobile-nav.md` for the full rules. Summary: `Layout.tsx` holds both nav bars in the DOM at once; `useLongPress.ts` + `BottomSheet.tsx` port the old HTML app's long-press-opens-an-action-sheet and drag-to-dismiss gestures, as an *added* affordance alongside (not replacing) always-visible Edit/Delete buttons (icon buttons on mobile, labelled ones on desktop).

## Setup and Run

```bash
cd client
pnpm install
pnpm dev     # Vite dev server on :5173, proxies /api/* to http://localhost:8080 (override with VITE_API_PROXY_TARGET)
pnpm lint    # oxlint
pnpm build   # tsc -b && vite build, output in dist/
```

Run `pnpm lint` and `pnpm build` from inside `client/` before committing a change that touches it.

## Deploy Notes

- Production target is Vercel, with the project's Root Directory set to `client` (`client/vercel.json` holds the build command, output directory, and the SPA rewrite to `index.html`). Nothing outside `client/` is needed to build it, so there is no "include files outside the root directory" setting to enable.
- `VITE_API_BASE_URL` (the Render API's origin) is read in `lib/api/client.ts`, `lib/api/import.ts`, and `AuthContext.tsx`. Vite inlines `VITE_*` values at **build** time, so changing it needs a redeploy, not just a saved env var. Unset (local dev) means relative URLs through the dev proxy.
- The deployed origin must be listed in the server's `CORS_ALLOWED_ORIGINS`, and is also what the server's `APP_BASE_URL` should be — email links open `client/`'s `/reset-password` and `/verify-email` routes. See `.claude/rules/deployment.md`.

## Files Agents Should Prefer Reading Before Edits

- `client/src/App.tsx` — the actual route table and provider tree.
- `client/src/lib/api/types.ts` — the TypeScript shapes this client expects back from the API; check these against `server/internal/api`'s actual DTOs if anything looks off.
- `client/src/lib/api/client.ts` — the token-attach/refresh-retry logic every authenticated request goes through.
- `client/src/lib/auth/AuthContext.tsx` — the auth bootstrap and the `status` values `ProtectedRoute` gates on.
- `client/src/index.css` — the full CSS variable palette, both light and dark.
- `.claude/rules/*.md` — area-specific rules, auto-loaded for the file being touched.

## Safe Edit Rules For Agents

- Never persist the access token to `localStorage`/`sessionStorage` — it stays in `tokenStore.ts`'s module variable only. If you need to "fix" a reload losing it, the actual fix is checking the silent-refresh bootstrap in `AuthContext.tsx`, not adding persistence.
- A page with its own filter/view state that should be bookmarkable or linkable belongs in the URL (`useSearchParams`), not local `useState` — see Important Reality Checks above.
- When adding a mutation, check what else its write could affect (another resource's cached list, the dashboard's totals) and invalidate those query keys too — don't assume invalidating the mutation's own resource is enough.
- Never embed a raw formatted string from the API as if it were final display text without checking `.claude/context/server.md`'s "ship raw numbers" convention — if a field looks like it should already be formatted money/a date, it probably isn't, and `lib/format.ts` is where the formatting belongs.
- Comment sparingly: only what the code can't say (a hidden constraint, a non-obvious reason). One `//` line at most; anything longer goes in a single `/* */` block, never a stack of `//` lines. Otherwise there is no separate TypeScript style guide, so match the structure of the surrounding file.
- Verify an auth, cookie, or response-shape change in a real browser, not just `tsc`/`oxlint`/a visual read of the code — several real bugs here (the reload-loses-session cookie issue, the nil-slice-to-null JSON issue, a URL-sync issue) were invisible to static checks and only surfaced once something actually ran in Chromium. See `.claude/rules/json-api-conventions.md`'s browser-testing note.

## Common Task Playbooks

- Add a new authenticated page: add the route in `App.tsx` (lazy-loaded, inside the `ProtectedRoute`/`Layout` nesting), add its API functions to `lib/api/<resource>.ts` and types to `lib/api/types.ts`, add a TanStack Query hook in `hooks/use<Resource>.ts`, write the page component in `pages/`.
- Add a new mutation: add it to the resource's `hooks/use<Resource>.ts`, and invalidate every query key the write could affect, not just its own resource's.
- Add a new CSS color token: add the `--c-*` variable to `:root` and to both dark palette blocks in `index.css` (the `@media (prefers-color-scheme: dark) :root:not(.light)` block and the `:root.dark` block), then map it in the `@theme` block.
- Add a form field: wrap the control in `components/ui/Field` (it supplies the label, the ids, `aria-invalid` and `aria-describedby`), and build buttons from `buttonClass()` rather than a new class string.
- Change what the API returns for an existing endpoint: update `lib/api/types.ts` to match, and grep for every call site that destructures the changed field before assuming a type-only change is safe (TypeScript won't catch a field that's optional on both sides but means something different now).

## Known Gaps and Debt

- Parts of the design handoff's motion spec aren't built: the inline-edit row doesn't animate its height (it fades in), there's no post-save row flash, no View Transitions, and dialogs close without an exit fade. The bottom sheet's drag thresholds stay at the existing 25% / 40px-in-250ms rather than the handoff's 30% / 0.5px/ms (`.claude/rules/mobile-nav.md`).
- The dark-mode chart colors (`#818CF8`, `#34D399`) sit slightly above the lightness band the dataviz palette validator wants for a dark surface. They pass contrast and colorblind separation, so they were kept as the handoff specified.
- No client-side email-verification reminder (see Important Reality Checks above).
- Settings' danger-zone card has no CSV export link above the delete button, which the deleted HTML app had (see `.claude/rules/account-deletion.md`); export is only on the Transactions page.
- No automated visual/accessibility regression testing — verification so far has been manual real-browser smoke testing (Playwright scripts run ad hoc, not checked into CI) plus `tsc`/`oxlint`/`vite build`.
- Bundle is route-split but not further optimized; `react-chartjs-2`/`chart.js` (~170KB) is the only chunk worth watching if it grows.

## Change Log

- `2026-10-07`: full UI redesign to the "Pocket Mono" handoff from Claude Design. `index.css` has a new semantic token set (light and dark), motion tokens and keyframes, and two Google Fonts (Be Vietnam Pro, Bricolage Grotesque); `lucide-react` added for icons. New `components/ui/` primitives (`Field`, `ConfirmDialog`, `Banner`, `EmptyState`, `PageSkeleton`, …), a toast context, and `buttonClass()` replacing `primaryButtonClass`; `FieldError` deleted. Dashboard and Categories became folders like the other pages, and `AuthPage` split into `AuthTabs`/`LoginForm`/`RegisterForm`. Behavior changes: the dashboard charts now follow the theme (they stayed light before); `confirm()`/`alert()` replaced by `ConfirmDialog`; deleting an income category that has transactions explains why it can't (the server refuses it) instead of offering a delete that fails; the month picker is a keyboard listbox; Transactions gained min/max amount filters, removable filter chips and "Clear filters"; the add form's default date is the local date (it was UTC, so yesterday before 7am in Vietnam); renaming a category now refreshes transactions and the dashboard too. Checked in Chromium at 1280px and 390px, light and dark, keyboard and long-press included.
- `2026-10-06`: every page now shows an error when its query fails, instead of "Loading…" forever (it checked `isLoading || !data` before `error`, which TanStack Query v5 never reaches); the `react-component` skill's Loading section describes the new shape. Transactions got an accessibility pass: labels on the add form, `aria-label`s on the filter bar and the inline edit form, `aria-pressed` on the expense/income toggle, row-specific names on Edit/Delete, focus moved into the edit form and back to Edit on save/cancel/Escape, and a failed delete shown inline instead of in `alert()`. Checked in Chromium (desktop, and mobile in dark mode).
- `2026-10-06`: `TransactionsPage`, `ImportPage` and `SettingsPage` (430, 339 and 274 lines) each split into a folder under `pages/`, one file per sub-component, with no behavior change. The extracted components take `Category[]`/`Session[]` instead of restating those shapes inline. `App.tsx`'s lazy imports and the `paths:` of `req-value-objects.md`, `csv-import.md` and `mobile-nav.md` follow the new paths.
- `2026-10-06`: `AuthLayout.tsx` slimmed to the shell itself. `FieldError` moved to `components/FieldError.tsx` and `inputClass`/`primaryButtonClass` to `lib/formStyles.ts`, since Settings and Import (pages behind auth) were importing form styles from the pre-auth layout. Frontend Layout now says `components/layout/` holds both app shells, which `AuthLayout` already did.
- `2026-10-06`: comments across `client/` cut to the ones the code can't speak for. History and pointers to the deleted HTML app are gone, and anything longer than one line is now a single `/* */` block. The Safe Edit Rules and the `react-component` skill now state that convention.
- `2026-10-06`: `.claude/skills/react-component/` added, covering where a component/page/hook file goes, how props/state/async states are written, and an accessibility pass, with patterns in its `references/accessibility.md`. Read This First no longer lists individual rule files (see `.claude/context/README.md`'s Maintenance Rule).
- `2026-10-05`: pnpm workspace removed. `pnpm-workspace.yaml` deleted and `pnpm-lock.yaml` moved from the repo root into `client/` (regenerated as a standalone lockfile, same resolved versions); `packageManager` added to `client/package.json`; CI points at `client/pnpm-lock.yaml`. `client/` is now a plain standalone package. Added Setup and Run and Deploy Notes sections to this file.
- `2026-10-05`: this file renamed from `frontend.md` to `client.md` to match the `client/` directory it describes.
- `2026-10-05`: the Gin/React migration completed. The old `html/template` + htmx frontend (`server/internal/web`, the `view_*.go` render pipeline, `server/internal/csrf`) was deleted in full. This file rewritten from scratch to describe `client/`'s React SPA as the frontend, replacing the previous version which described the now-deleted templates/static-asset system.
- `2026-10-05`: `client/` built out across several commits to full feature parity with the deleted HTML app (auth, all four main pages, CSV import UI, mobile gestures, code-splitting) — see git log on this date for the detailed, phase-by-phase history; this file no longer tracks that transition day by day now that it's finished.
- `2026-10-05`: `client/` scaffolded (Vite + React + TypeScript + Tailwind v4 + pnpm), as part of converting the repo into a monorepo (`server/` + `client/`).
