# Accessibility patterns for $pend's client

Code patterns for the checklist in `SKILL.md` section 3. Each one is short. Copy the shape and keep the neighbors' class names.

## Contents
1. Labeled field with an error
2. Compact field with no room for a visible label
3. Row actions in a list
4. Inline edit: focus in, focus back
5. Dialogs and bottom sheets
6. Toggle groups and tab-like links
7. Status and error messages
8. Images, icons, and decorative marks

## 1. Labeled field with an error

Generate ids with `useId()` rather than writing `id="amount"`. Every `TransactionRow` renders its own edit form, so a fixed id would collide.

```tsx
const amountId = useId()
const amountErrorId = useId()

<label htmlFor={amountId} className="mb-1 block text-[13px] text-ink-muted">Amount</label>
<input
  id={amountId}
  type="number"
  inputMode="numeric"
  required
  min={1}
  aria-invalid={amountError ? true : undefined}
  aria-describedby={amountError ? amountErrorId : undefined}
  ...
/>
{amountError && <p id={amountErrorId} className="mt-1.5 text-[13px] text-danger">{amountError}</p>}
```

Prefer native constraints (`required`, `min`, `type="email"`) over hand-written checks. The browser announces those for free. Write a custom message only for rules the browser can't express ("Please choose a category.").

## 2. Compact field with no room for a visible label

In the filter bar on Transactions, a visible label for every control would double the bar's height. Give each control an accessible name instead. Keep the placeholder as a visual hint only.

```tsx
<input aria-label="Search transactions" placeholder="Search…" ... />
<select aria-label="Transaction type" ...>
```

A `<select>` whose first option is "All categories" still needs `aria-label`. The selected option's text is the control's *value*, not its *name*.

## 3. Row actions in a list

Visible text can stay short. The accessible name has to say which row the action is for.

```tsx
const rowName = transaction.description || transaction.categoryName

<button type="button" onClick={() => setEditing(true)} aria-label={`Edit ${rowName}`}>Edit</button>
<button type="button" onClick={() => void onDelete()} aria-label={`Delete ${rowName}`}>Delete</button>
```

Start the `aria-label` with the visible word ("Edit …"). Voice-control users say what they see, and the label has to contain it for "click Edit" to work.

## 4. Inline edit: focus in, focus back

When a row swaps into an edit form, the button the user just pressed unmounts and focus falls to `<body>`. A keyboard user is then back at the top of the page. Fix both directions:

```tsx
const editButtonRef = useRef<HTMLButtonElement>(null)
const wasEditing = useRef(false)

// The button only exists again after the re-render, so focus it from an effect.
useEffect(() => {
  if (wasEditing.current && !editing) editButtonRef.current?.focus()
  wasEditing.current = editing
}, [editing])

// In the edit form, the first field takes focus on mount:
<select autoFocus ...>

// In the read-only row:
<button ref={editButtonRef} type="button" ...>Edit</button>
```

Escape should cancel the edit too, matching what a dialog does:

```tsx
<form onSubmit={onSave} onKeyDown={(e) => { if (e.key === 'Escape') setEditing(false) }}>
```

## 5. Dialogs and bottom sheets

Use `components/BottomSheet.tsx` (a real `<dialog>` opened with `showModal()`) rather than building a new overlay. The native dialog traps focus, closes on Escape, makes the rest of the page inert, and returns focus to the element that opened it. A `<div>` overlay has to reimplement all of that, and it usually gets something wrong.

What's still on you:
- Give the sheet's content a heading or a leading line of text that says what it's for. `TransactionRow`'s sheet shows the transaction's description.
- Its buttons are real `<button type="button">` elements.
- Closing it from an action (Edit, Delete) calls `onClose` before starting the action, so focus handling runs in order.

Ask before a delete with `components/ui/ConfirmDialog`: a native `<dialog>` with initial focus on Cancel, named by its title and described by its body. Show a failed mutation's error inline (in the dialog, next to the row or form), never in `alert()`.

## 6. Toggle groups and tab-like links

A set of buttons where one is "on" (expense/income in the add form, the theme picker): use `aria-pressed` on each, and wrap the set in `role="group"` with a name.

```tsx
<div role="group" aria-label="Transaction type" className="flex gap-1 rounded-[14px] bg-surface-2 p-1">
  {(['expense', 'income'] as const).map((t) => (
    <button key={t} type="button" aria-pressed={type === t} onClick={() => setType(t)} ...>{t}</button>
  ))}
</div>
```

Tab-looking links that actually navigate (`AuthPage`'s Log in / Sign up, the nav bars) are links, not tabs. Mark the current one with `aria-current="page"`. react-router's `NavLink` sets this automatically; a plain `Link` doesn't. Don't add `role="tab"` to a link, because a tab promises arrow-key navigation a link doesn't have.

## 7. Status and error messages

```tsx
<p role="status" className="text-ink-muted">Loading…</p>              // polite, announced when it appears
<p role="alert" className="text-[13px] text-danger">{error}</p>      // assertive, interrupts
```

Screen readers announce a live region when its *content changes*. `role="alert"` is the forgiving one: conditionally rendering `{error && <p role="alert">…</p>}` is announced reliably, even though the element and its text arrive together. `role="status"` is less dependable when it's inserted already filled in. For a status that matters (e.g. "Imported 42 transactions"), render an empty `role="status"` container up front and change its text later.

## 8. Images, icons, and decorative marks

- Purely decorative (the category color dot next to the name, `BottomSheet`'s grab handle): `aria-hidden="true"`.
- Conveys meaning with no text beside it: `role="img"` plus `aria-label` (as `AuthLayout`'s wordmark does), or a visually hidden text span.
- Icon-only buttons (if icons come back to the mobile nav): `aria-label` on the `<button>`, and `aria-hidden="true"` on the `<svg>`.
- Charts (`DashboardPage`): the canvas is opaque to a screen reader. Put the same numbers in readable text nearby, a legend list or a summary sentence, and give the `<canvas>` `role="img"` plus an `aria-label` that summarizes it.
