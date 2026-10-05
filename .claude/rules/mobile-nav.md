---
paths:
  - "client/src/components/layout/Layout.tsx"
  - "client/src/components/BottomSheet.tsx"
  - "client/src/hooks/useLongPress.ts"
---

# Mobile navigation and gestures

## Layout (`Layout.tsx`)

- Both nav bars exist in the DOM at once -- a desktop `<nav>` and a mobile `<header>`/bottom `<nav>` -- each hidden at the breakpoint the other owns via Tailwind's `md:` prefix, rather than mounting/unmounting one on resize. This mirrors `nav.html`'s original `nav_desktop`/`nav_mobile` split.
- The mobile bottom bar is icon-free right now (text labels only) -- a simplification from the original SVG-icon bar, not a deliberate redesign. Add icons back if you pick this up.
- `useDashboard()` is called once at the top of `Layout`, shared by every page below it (see `.claude/rules/balance-widget.md`) -- there is exactly one request for `headerBalance`, not one per page.

## Long-press action sheet (`useLongPress.ts`, used in `TransactionsPage.tsx`'s `TransactionRow`)

- Ports `server/internal/web/static/app.js`'s long-press IIFE: pointer events (not a touch/mouse pair) unify both input kinds into one set of handlers, a ~500ms hold fires the callback, and a move past `moveTolerance` (10px default) cancels it the same way a scroll does there.
- It's an *added* affordance, not a replacement: `TransactionRow` keeps its always-visible Edit/Delete text buttons for a mouse/keyboard user or anyone who never discovers the gesture, and the long-press opens a `BottomSheet` offering the identical two actions.

## Bottom sheet (`BottomSheet.tsx`)

- A real `<dialog>` (`showModal()`), not a styled `<div>` -- native focus trapping, Escape-to-close, and a `::backdrop` with no extra markup, the same reason the original HTML app used one.
- Drag the grab handle down to dismiss: the sheet follows the pointer (downward only -- dragging up must not lift it off the bottom), and on release either snaps back or slides out, based on the exact same two-branch threshold the original used -- past a quarter of the sheet's height, or a short flick (>40px in under 250ms). Keep both branches and both numbers in sync if you touch either implementation; `server/internal/web/static/app.js`'s version was deleted in the migration's cleanup, so this file is now the only copy of that logic, not a port to keep in sync with a surviving original.
- A click on the dialog element itself (never a descendant -- the sheet's own content box is the only thing covering any of the viewport other than the backdrop) is treated as a backdrop click and closes it.
