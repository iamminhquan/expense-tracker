---
paths:
  - "internal/csvimport/**"
  - "internal/handlers/import_*.go"
  - "internal/web/templates/import.html"
  - "internal/handlers/txn_export.go"
---

# CSV import

`internal/csvimport` with `import_handlers.go` and `import_mapping.go` reads any CSV that has one transaction per row.

## Model

- What a column means is a `Mapping`: which column plays which role, the order of the date parts, whether a minus sign marks an expense. The format the app exports is one such Mapping (`ExportMapping`), recognised by `Sniff`, so a round trip skips the mapping screen.
- `csvimport` never touches the database. The account arrives as a `[]csvimport.Category` and the answer leaves as a `Sheet` or an `Import`. Keep it that way so every rule about reading a file is testable without Postgres.
- `Sniff` proposes, `Plan` applies. `Plan` does not judge headers: a mapping that cannot reach a line reports that line instead of refusing the file.

## Guessing

- Guess by header name first (`headerAliases`, which spells out toned and untoned Vietnamese rather than carrying a Unicode normaliser), then by content. A column is a date or an amount if `contentShare` of it parses as one; of the columns left, the one that repeats most is the category and the one that repeats least is the note.
- Every guess is rendered into a control the user can change, which is what licenses rules this rough.
- Date order is the exception. A column whose days never pass the 12th fits both DD/MM and MM/DD, and it is the only wrong guess that still produces rows that look right. `Sheet.AmbiguousDate` makes the screen say so, and a preview whose failures are mostly date failures says the format is probably wrong instead of listing hundreds of complaints.

## Amounts

- `parseAmount` strips currency symbols, spaces and accounting parentheses, and resolves `.` versus `,` by position. With both present, the last one is the decimal point; with one present, three digits after it means thousands. "45.000" is forty-five thousand.
- Fractions round to whole đồng, and the preview says how many rows were rounded. Refusing them would mean a file in a currency with cents imports nothing.

## Categories

- Resolve a category name against the defaults through `i18n` and the slug, never the `name` column (see `categories.md`).
- A name that matches nothing is planned as a new personal category, one per (name, type) to match the table's uniqueness. The preview names them before anything is written.
- `csvimport.MatchKey` is exported because the handler that pairs rows with the categories it just created must use the same rule. Two subtly different rules would leave a row pointing at a category that was never made.

## Behaviour

- Import is all-or-nothing: one bad line blocks the file. A partial import would mean fixing three lines and re-importing a file whose other 197 are already in, and nothing in the schema can tell the second copy apart.
- Exact duplicates are counted and reported, not refused (`countImportDuplicates`, one query over the file's date range). Two identical coffees on one day are real.
- All three steps are one handler with no server-side state. The upload form keeps the file in the DOM and every step re-sends it through `hx-include`, so the mapping travels as form fields. The file is sniffed on every request, then rewound and planned.
- `Import.Fingerprint` is a digest of what was read, echoed in a hidden field, so a file swapped between steps is refused rather than imported unseen.
- Row validation enforces what the quick-add form enforces (amount, type, note length, future limit). Don't add a laxer way in.
- The two numeric limits live in `internal/txnrule` and are read from there by the form, the inline edit and the importer. Never copy the numbers.
