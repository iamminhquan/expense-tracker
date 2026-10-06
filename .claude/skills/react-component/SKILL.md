---
name: react-component
description: Write, split, or refactor React components, pages, and hooks in $pend's `client/` the way a large codebase needs them — where each file goes, how props/state/async states are written, and an accessibility pass. Use whenever you create or edit anything under `client/src/pages/`, `client/src/components/`, or `client/src/hooks/`: a new page or route, a form, a list row, a modal or bottom sheet, extracting sub-components from an oversized page, or fixing a component's structure or accessibility — even when the request never says "component".
---

# Writing React Components in $pend

`client/` is a Vite + React 19 + TypeScript + Tailwind v4 SPA with TanStack Query and react-router. "Large-project standard" here means four things: every file has one obvious home, every component has one job, server data flows through exactly one layer, and the UI works for a keyboard or screen-reader user, not just a mouse. Most of this already holds in the existing code. Match it, and fix the known weak spots called out below instead of copying them.

## Workflow

1. **Read before writing.** Open `client/src/App.tsx` (routes, provider tree), the files next to where you're working, `lib/api/types.ts` for the data shapes, and the `.claude/rules/*.md` whose `paths:` cover the files you're touching. Those load on their own when you open a matching file. To see every frontend rule up front, run `grep -l '^  - "client/' .claude/rules/*.md`.
2. **Decide placement** (section 1).
3. **Write the component** (section 2).
4. **Do the accessibility pass** (section 3; patterns in `references/accessibility.md`).
5. **Verify** (section 4).

## 1. Where the file goes

| What you're writing | Where it lives |
|---|---|
| A route-level screen | `pages/<Name>Page.tsx`, or `pages/<area>/<Name>Page.tsx` once it has its own sub-components (see below). Pre-auth screens live in `pages/auth/`. |
| A sub-component only one page uses | Start inside that page's file; move to a sibling file in the page's folder when it outgrows the file (below). |
| A component two or more pages use | `components/<Name>.tsx` |
| A design-system primitive (a field, button-like control, badge, banner, dialog, empty/loading state) | `components/ui/<Name>.tsx`. Check what's already there before adding one |
| A piece of the authenticated shell (nav, header widgets, route guards) | `components/layout/` |
| Server data: queries + mutations | `hooks/use<Resource>.ts`, calling `lib/api/<resource>.ts`, typed by `lib/api/types.ts` |
| A reusable non-data behavior | `hooks/use<Behavior>.ts` (like `useLongPress.ts`) |
| App-wide client state | A context in `lib/<area>/<Name>Context.tsx` — rare; the TanStack cache plus the URL already cover most of what a global store would hold |
| Pure helpers (formatting, parsing) | `lib/` (e.g. `lib/format.ts`) |

**When a page becomes a folder.** Once a page file passes roughly 250 lines, or holds three or more sub-components, turn it into a folder: `pages/transactions/TransactionsPage.tsx` plus `FilterBar.tsx`, `AddTransactionForm.tsx`, `TransactionRow.tsx` beside it. Update the lazy import path in `App.tsx`. `pages/transactions/`, `pages/import/` and `pages/settings/` are already laid out this way; `CategoriesPage.tsx` and `auth/AuthPage.tsx` are the next closest, so split them as part of any real change that grows them.

**Promote late.** A component moves up to `components/` only when a second page actually needs it. Don't do it because it "might be reused." A shared component built too early collects a prop for every caller's special case. A page-local one stays small because it serves one caller.

**File and export conventions** (these match the existing code and tooling, so keep them):
- One exported component per file, file name = component name in PascalCase. oxlint's `react/only-export-components` and Vite fast refresh both depend on this. Exporting a plain constant alongside is fine, but a constant other files import belongs in `lib/` (`lib/formStyles.ts`), not in whichever component happened to define it first.
- Named exports: `export function TransactionRow(...)`. `App.tsx`'s lazy routes do `.then((m) => ({ default: m.XPage }))`, so a page with a default export breaks that pattern.
- A new page gets a `lazy()` route in `App.tsx`, inside the `ProtectedRoute` → `Layout` nesting if it needs auth. Each route is its own chunk.
- Hooks are `useCamelCase.ts`.
- No barrel `index.ts` files. They hide where a component actually lives, and a barrel that re-exports a heavy module can pull it into a chunk that never needed it.
- Imports are relative (`../../hooks/useTransactions`). There's no `@/` alias configured.

## 2. Writing the component

### Props and types
- An exported component declares `interface <Name>Props` right above it and destructures in the signature (see `MonthPicker.tsx`, `BottomSheet.tsx`). A small file-local sub-component can use an inline object type.
- Reuse the API types instead of restating them. Write `categories: Category[]` or `Pick<Category, 'id' | 'name'>[]`, not `{ id: number; name: string; type: string }[]`. Restating the shape inline is the pattern to avoid: when the DTO changes, the compiler should point at every caller.
- Document a prop with `/** ... */` only when *why* it exists isn't obvious (see `MonthPicker`'s `allowAllMonths`).
- Callback props are `on<Event>` (`onChange`, `onClose`). Handlers inside a component follow the existing `onSubmit` / `onSave` / `onDelete` naming.
- `verbatimModuleSyntax` is on, so type-only imports need `type`: `import { useState, type FormEvent, type ReactNode } from 'react'`. Prefer that over the global `React.FormEvent`.

### One job per component
A page component is the orchestrator. It reads the URL and the queries, handles the page-level loading/error state, and composes sections, passing data down. Each section (filter bar, add form, list row) owns its own form state and its own mutations. Split a component when its JSX passes roughly 150 lines, or when it holds two pieces of state that never interact (a list and an unrelated add form).

### State: put each kind where it belongs
- **Server data** comes only from a `hooks/use*` hook. Never call `fetch` or `lib/api/*` from a component, and never copy query data into `useState`. The one exception is seeding an edit form's initial values, as `TransactionRow` does. That works because the row is keyed by `transaction.id` and remounts for a different entity.
- **View state worth linking to or reloading into** (filters, month, page number, active tab) lives in the URL via `useSearchParams`. It does not live in `useState`. A real bug came from exactly this: a link to `/transactions?month=2026-02` silently showed the current month. See `.claude/rules/req-value-objects.md`.
- **Form inputs and UI toggles** are local `useState`, kept as low in the tree as the state's users allow.
- **Derived values** are computed during render (`const options = categories.filter(...)`). Don't keep them in `useState` and sync them with `useEffect`. Reach for `useMemo` only when the computation is measurably expensive.
- **`useEffect`** is for syncing with something outside React: `dialog.showModal()`, a class on `<html>`. It is not for reacting to your own state changes. Do that work in the event handler that changed the state.

### Loading, error, empty
Branch on `data` first, then tell an error apart from a load in progress:

```tsx
const { data, error, refetch } = useTransactions(filters)
if (!data) {
  if (error) return <InlineError message="Could not load transactions." onRetry={() => void refetch()} />
  return <PageSkeleton label="Loading transactions…" />
}
```

In TanStack Query v5 a failed query has `isLoading === false` and `data === undefined`, so the old `if (isLoading || !data) return <Loading/>; if (error) ...` never reached the error branch: a failed request showed "Loading…" forever. Checking `error` before `data` is wrong the other way: a background refetch that fails keeps the cached `data` but sets `error`, and the page would throw away what it was showing. Every page follows the shape above; keep it.

`InlineError` and `PageSkeleton` live in `components/ui/`; `PageSkeleton` waits 150ms before drawing, so a fast response never flashes a skeleton.

Empty is its own state, and it should say *why* it's empty. `TransactionsPage` tells "No transactions in October" apart from "Nothing matches your filters" (both through `components/ui/EmptyState`). Those two need different responses from the user.

### Mutations
- Call `mutation.mutateAsync(...)` inside `try`/`catch`, and show `err instanceof ApiError ? err.message : 'Could not <verb> the <thing>.'` next to the form. `ApiError` comes from `lib/api/client.ts`. The server's messages are written for users.
- Confirm success with `useToast().success(...)` (`lib/toast/ToastContext.tsx`) when the change isn't already visible where the user is looking. A destructive action asks first through `components/ui/ConfirmDialog`, never `window.confirm()`.
- Disable the submit button with `disabled={mutation.isPending}` and mark it `aria-busy`.
- Cache invalidation lives in the hook, never in the component. A new mutation in `hooks/use<Resource>.ts` invalidates every query key its write can change, not just its own. `useTransactions.ts`'s `invalidateEverythingATransactionTouches` is the pattern.

### Styling
- Use Tailwind utilities against the theme tokens, chosen by role: `bg-app`, `bg-surface`, `bg-surface-2`, `border-border`, `border-border-strong`, `text-ink`, `text-ink-muted`, `bg-accent`/`text-on-accent`, `text-expense`, `text-income`, `text-danger`, `text-warning`, and the `*-tint` fills behind them. Headings and money figures use `font-display`; amounts, dates and counts add `tabular`. Never use a hex or rgb value, in a class or in `style`. If a new color is needed, add it as a token to `index.css`, in both dark-palette blocks (`.claude/rules/theming.md`). Otherwise it breaks dark mode. The one exception is user data that is itself a color, like `transaction.categoryColor`.
- Match the neighbors' scale. The app uses explicit sizes (`text-[15px] leading-[22px]`, `rounded-[12px]` controls, `rounded-[28px]` cards) rather than Tailwind's named steps. Controls and buttons are 44px tall (`h-11`); a touch target is never smaller.
- Build conditional classes with a template literal, as the existing code does. There's no `clsx` dependency; don't add one for a single use.
- Reuse before writing: buttons come from `buttonClass(variant, size)` in `lib/formStyles.ts` (with `inputClass`, `selectClass`, `cardClass`, `pageTitleClass` beside it), and fields from `components/ui/` (`Field`, `SelectControl`, `AmountInput`, `PasswordInput`, `SegmentedControl`, `Checkbox`, `Badge`, `Banner`). When a new long class string appears three or more times, hoist it the same way, so the next change to it happens in one place.
- Design for both breakpoints. Mobile first, with `md:` for desktop (see `Layout.tsx`). When the two layouts differ in structure rather than spacing (a table row on desktop, a stacked card on mobile), pick one with `useIsDesktop()` instead of rendering both and hiding one, so a screen reader never meets duplicate controls. A touch gesture like long-press is an *added* way to reach an action. The same action must also be reachable through a visible button (`.claude/rules/mobile-nav.md`).

### Lists and performance
- `key` is the entity's id, never the array index. Index keys make React reuse a row's state (like an open edit form) for a different item after a delete or re-sort.
- Don't add `React.memo`, `useCallback`, or `useMemo` by default. They make the code harder to follow and pay off only for a measured slow render. Route-level `lazy()` is the code-splitting this app uses.

### Comments
Write a comment only when the code can't say it: a constraint that isn't visible (the token never goes to `localStorage`), a reason something that looks wrong is right, a workaround. Don't restate the code, narrate its history, or point at files that no longer exist. Most components need none.

- Keep a comment to one `//` line of about 90 characters or less.
- If it truly needs more, use one `/* */` block. Never stack several `//` lines.
- Use `/** */` on a prop only when its name doesn't already explain it.

## 3. Accessibility pass

Go through these for every component you write or touch. `references/accessibility.md` has the code patterns for each one. Read it when you're building a form, a dialog, a toggle group, or a list with row actions.

- **Use the native element.** `<button type="button">` for actions, `<Link>` for navigation, `<form onSubmit>` so Enter submits, `<dialog>` with `showModal()` for anything modal (as `BottomSheet` and `ConfirmDialog` do). A clickable `<div>` has no keyboard support and no role.
- **Every input has a label.** Use a visible label: `components/ui/Field` renders one and wires the id from `useId()` (a component can render more than once on a page, and two `id="amount"` inputs collide). A placeholder is not a label: it disappears as soon as the user types. When a compact layout (the filter bar) truly has no room for a visible label, `aria-label` is the minimum. `pages/transactions/` shows both: visible labels in `AddTransactionForm`, `aria-label`s in `FilterBar` and the desktop inline edit row.
- **Repeated row actions need context.** A list of rows that each have "Edit" and "Delete" gives a screen reader ten identical "Delete" buttons. Name them with what they act on: `aria-label={\`Delete ${transaction.description || transaction.categoryName}\`}`. An `aria-label` that just repeats the visible text (`aria-label="Edit"` on a button reading "Edit") adds nothing.
- **Errors are announced and attached to their field.** A form-level error goes in `role="alert"`. A field error sets `aria-invalid` and is linked with `aria-describedby`.
- **Loading text uses `role="status"`**, so it's announced without stealing focus.
- **Toggle buttons expose their state** with `aria-pressed` (`components/ui/SegmentedControl` does this for the type toggles and `UserMenu`'s theme switch).
- **Color is never the only signal.** Expense vs. income is also carried by the sign (`formatVNDSigned`). A category color dot sits next to the category's name.
- **Focus stays visible and stays put.** `index.css` gives every `:focus-visible` element a 3px `accent` outline; never remove it. When an inline edit form opens, move focus into its first field. When it closes, focus should land back on the button that opened it.
- **Headings follow the page.** One `<h1>` per page, then `<h2>` for its sections.

## 4. Verify

From inside `client/`:

```bash
pnpm lint    # oxlint: rules-of-hooks, only-export-components
pnpm test    # vitest
pnpm build   # tsc -b && vite build
```

Then:
- **Keyboard-only pass** for anything interactive. Tab through it, use Enter/Space on every control, and use Escape on dialogs. Check that focus is visible at each stop and never gets lost.
- **Real-browser check** for anything that touches auth, a response shape, the URL, or a gesture. Several real bugs here were invisible to `tsc`/`oxlint` and only showed up in Chromium (`.claude/context/client.md`, Safe Edit Rules). Check light and dark mode, and a narrow mobile viewport.
- **Docs.** If you added a folder, moved a page into one, or changed a convention above, update `.claude/context/client.md`'s Frontend Layout section in the same change.
- **Rule paths.** If you moved or renamed a file, run `grep -rn '<old path>' .claude/`. Every rule's `paths:` entry that still names the old path stops loading silently, so fix those in the same change.

## Example: a page section done right

```tsx
import { useState, type FormEvent } from 'react'
import { Plus } from 'lucide-react'
import { useCreateCategory } from '../../hooks/useCategories'
import { ApiError } from '../../lib/api/client'
import { buttonClass, inputClass } from '../../lib/formStyles'
import { useToast } from '../../lib/toast/ToastContext'
import { Field } from '../../components/ui/Field'
import type { Category } from '../../lib/api/types'

interface QuickCategoryFormProps {
  type: Category['type']
  /** One of the swatches; picked by the parent's color picker. */
  color: string
}

export function QuickCategoryForm({ type, color }: QuickCategoryFormProps) {
  const createCategory = useCreateCategory()
  const toast = useToast()
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      await createCategory.mutateAsync({ name, type, color })
      toast.success(`Added "${name}"`)
      setName('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not add the category.')
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <Field label={`New ${type} category`} error={error}>
        {(control) => <input {...control} required value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />}
      </Field>
      <button type="submit" disabled={createCategory.isPending} aria-busy={createCategory.isPending} className={buttonClass('primary')}>
        <Plus aria-hidden="true" />
        Add
      </button>
    </form>
  )
}
```

What it shows: a props interface built from the API type, the mutation coming from the hook (which owns invalidation), `Field` supplying the label, the ids and the error wiring (`aria-invalid`, `aria-describedby`, the message itself), a toast on success, a submit button disabled while pending, and no colour or size invented on the spot. The hook and input type are the real ones (`useCreateCategory`, `CreateCategoryInput` in `lib/api/categories.ts`). The component itself is illustrative: `pages/categories/AddCategoryForm.tsx` is the real add form.
