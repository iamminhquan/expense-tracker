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
- The mobile bottom bar is a floating pill with an icon and label per tab (`lucide-react`), and a separate round "+" button to its right, in thumb reach. The button opens `AddTransactionSheet`, mounted on its first tap so the categories request isn't made on every page load. Desktop has no such button: the same `AddTransactionForm` is inline on the Transactions page, and the form picks its layout with `useIsDesktop()`. On mobile the header has no room for the balance widget, so `UserMenu` shows it at the top of its panel instead; the mobile header sits at `z-50` so the menu's scrim covers the tab bar too.
- `useDashboard()` is called once at the top of `Layout`, shared by every page below it (see `.claude/rules/balance-widget.md`) -- there is exactly one request for `headerBalance`, not one per page.

## Long-press action sheet (`useLongPress.ts`, used in `pages/transactions/TransactionRow.tsx`)

- Ports `server/internal/web/static/app.js`'s long-press IIFE: pointer events (not a touch/mouse pair) unify both input kinds into one set of handlers, a ~500ms hold fires the callback, and a move past `moveTolerance` (10px default) cancels it the same way a scroll does there.
- It's an *added* affordance, not the only way in: on mobile each `TransactionRow` has a visible 44px "…" button (`Open actions for <row>`) that opens the same `BottomSheet` as the long-press, with Edit and Delete; desktop rows (chosen with `useIsDesktop()`) keep labelled Edit/Delete buttons. A mouse, keyboard or screen-reader user never needs the gesture.

## Bottom sheet (`BottomSheet.tsx`)

- A real `<dialog>` (`showModal()`), not a styled `<div>` -- native focus trapping, Escape-to-close, and a `::backdrop` with no extra markup, the same reason the original HTML app used one.
- Drag the grab handle down to dismiss: the sheet follows the pointer (downward only -- dragging up must not lift it off the bottom), and on release either snaps back or slides out, based on the exact same two-branch threshold the original used -- past a quarter of the sheet's height, or a short flick (>40px in under 250ms). Keep both branches and both numbers in sync if you touch either implementation; `server/internal/web/static/app.js`'s version was deleted in the migration's cleanup, so this file is now the only copy of that logic, not a port to keep in sync with a surviving original.
- Closing slides the sheet out over 200ms before `dialog.close()` runs (instant under `prefers-reduced-motion`); a Delete chosen from the sheet then opens `components/ui/ConfirmDialog`, which on mobile is itself a bottom sheet but without drag. The design handoff proposed 30% / 0.5px/ms thresholds; they were not adopted, the numbers above stand.
- A click on the dialog element itself (never a descendant -- the sheet's own content box is the only thing covering any of the viewport other than the backdrop) is treated as a backdrop click and closes it.
