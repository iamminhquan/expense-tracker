---
paths:
  - "client/src/components/layout/Layout.tsx"
  - "client/src/components/BottomSheet.tsx"
  - "client/src/hooks/useLongPress.ts"
  - "client/src/pages/transactions/TransactionRow.tsx"
  - "client/src/components/AddTransactionSheet.tsx"
  - "client/src/components/AddTransactionForm.tsx"
---

# Mobile navigation and gestures

## Layout (`Layout.tsx`)

- Both nav bars exist in the DOM at once -- a desktop `<nav>` and a mobile `<header>`/bottom `<nav>` -- each hidden at the breakpoint the other owns via Tailwind's `md:` prefix, rather than mounting/unmounting one on resize. This mirrors the deleted HTML app's `nav_desktop`/`nav_mobile` split.
- The mobile bottom bar is a floating dock: Overview and Transactions, a raised magenta `+` in the middle, then Categories and Settings. The `+` opens `components/AddTransactionSheet.tsx` (a `BottomSheet` around `components/AddTransactionForm.tsx`) from any page, so logging a spend is a thumb action everywhere; on phones the transactions page has no inline add form. The sheet is rendered only when `!useIsDesktop()`. Pages clear the dock with `pb-[var(--dock-clearance)]`, and toasts sit above it the same way.
- The mobile header shows the balance widget in its compact form (ring and figure, the sentence for screen readers only) beside the account menu; the header sits at `z-50` so the menu's scrim covers the dock too.
- On phones the `MonthPicker` list opens fixed just above the dock rather than under its button, and the transaction filters open in a `BottomSheet` with labelled fields.
- `useDashboard()` is called once at the top of `Layout`, shared by every page below it (see `.claude/rules/balance-widget.md`) -- there is exactly one request for `headerBalance`, not one per page.

## Long-press action sheet (`useLongPress.ts`, used in `pages/transactions/TransactionRow.tsx`)

- Ports `server/internal/web/static/app.js`'s long-press IIFE: pointer events (not a touch/mouse pair) unify both input kinds into one set of handlers, a ~500ms hold fires the callback, and a move past `moveTolerance` (10px default) cancels it the same way a scroll does there.
- It's an *added* affordance, not a replacement: on phones every `TransactionRow` has a visible 44px "…" button that opens the same `BottomSheet` as the long-press (Edit / Delete; Edit swaps the sheet to `EditTransactionForm`'s stacked layout), and on desktop the row keeps labelled Edit/Delete buttons and edits inline. The row picks its layout with `useIsDesktop()`.
- After a long-press fires, the finger lifting still produces a `click`, and because the sheet's `showModal()` has just made the page inert, that click is retargeted to the `<dialog>` itself, which `BottomSheet` reads as a backdrop tap. `useLongPress` returns an `onClickCapture` that swallows exactly that one click; spread all of its handlers. A synthetic `pointerdown` never shows this; a real touch (CDP `Input.dispatchTouchEvent`) does.
- The sheet and the confirm dialog render inside the row's `<li>`, so presses in them bubble to the row's long-press handlers; `TransactionRow` ignores a long-press while either is open.

## Bottom sheet (`BottomSheet.tsx`)

- A real `<dialog>` (`showModal()`), not a styled `<div>` -- native focus trapping, Escape-to-close, and a `::backdrop` with no extra markup, the same reason the original HTML app used one.
- Drag the grab handle down to dismiss: the sheet follows the pointer (downward only -- dragging up must not lift it off the bottom), and on release either snaps back or slides out, based on the exact same two-branch threshold the original used -- past a quarter of the sheet's height, or a short flick (>40px in under 250ms). Keep both branches and both numbers in sync if you touch either implementation; `server/internal/web/static/app.js`'s version was deleted in the migration's cleanup, so this file is now the only copy of that logic, not a port to keep in sync with a surviving original.
- Closing slides the sheet out over 200ms before `dialog.close()` runs (instant under `prefers-reduced-motion`); a Delete chosen from the sheet then opens `components/ui/ConfirmDialog`, which on mobile is itself a bottom sheet but without drag. The design handoff proposed 30% / 0.5px/ms thresholds; they were not adopted, the numbers above stand.
- A click on the dialog element itself (never a descendant -- the sheet's own content box is the only thing covering any of the viewport other than the backdrop) is treated as a backdrop click and closes it.
