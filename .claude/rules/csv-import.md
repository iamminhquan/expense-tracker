---
paths:
  - "server/internal/csvimport/**"
  - "server/internal/api/import_handlers.go"
  - "server/internal/api/export_handlers.go"
  - "client/src/pages/ImportPage.tsx"
  - "client/src/lib/api/import.ts"
---

# CSV import

`server/internal/csvimport`, driven by `internal/api/import_handlers.go`, reads any CSV that has one transaction per row.

## Model

- What a column means is a `Mapping`: which column plays which role, the order of the date parts, whether a minus sign marks an expense. The format the app exports is one such Mapping (`ExportMapping`), recognised by `Sniff`, so a round trip skips the mapping screen.
- `csvimport` never touches the database. The account arrives as a `[]csvimport.Category` and the answer leaves as a `Sheet` or an `Import`. Keep it that way so every rule about reading a file is testable without Postgres.
- `Sniff` proposes, `Plan` applies. `Plan` does not judge headers: a mapping that cannot reach a line reports that line instead of refusing the file.
- `csvimport.DateFormat`, `NewCategory`, and `RowError` carry no JSON tags of their own -- `import_handlers.go` wraps each in a small DTO (`dateFormatDTO`, `importNewCategoryDTO`, `rowErrorDTO`) before it reaches a response. See `.claude/rules/json-api-conventions.md`.

## Guessing

- Guess by header name first (`headerAliases`, which spells out toned and untoned Vietnamese rather than carrying a Unicode normaliser), then by content. A column is a date or an amount if `contentShare` of it parses as one; of the columns left, the one that repeats most is the category and the one that repeats least is the note.
- Every guess is rendered into a control the user can change (`ImportPage.tsx`'s mapping screen), which is what licenses rules this rough.
- Date order is the exception. A column whose days never pass the 12th fits both DD/MM and MM/DD, and it is the only wrong guess that still produces rows that look right. `Sheet.AmbiguousDate` makes the screen say so, and a preview whose failures are mostly date failures (`mostlyDateErrors`) says the format is probably wrong instead of listing hundreds of complaints.

## Amounts

- `parseAmount` strips currency symbols, spaces and accounting parentheses, and resolves `.` versus `,` by position. With both present, the last one is the decimal point; with one present, three digits after it means thousands. "45.000" is forty-five thousand.
- Fractions round to whole đồng, and the preview says how many rows were rounded. Refusing them would mean a file in a currency with cents imports nothing.

## Categories

- Resolve a category name against the defaults through `i18n` and the slug, never the `name` column (see `categories.md`).
- A name that matches nothing is planned as a new personal category, one per (name, type) to match the table's uniqueness. The preview names them before anything is written.
- `csvimport.MatchKey` is exported because `applyImport` (which pairs rows with the categories it just created) must use the same rule. Two subtly different rules would leave a row pointing at a category that was never made.

## One endpoint, three outcomes

- `POST /api/v1/transactions/import` is upload, mapping, and preview collapsed into one endpoint a client drives by what it sends: no `mapped` field yet gets a `needsMapping` response (skipped entirely for a file shaped like this app's own export); a mapping without `confirm` gets a preview; `confirm=1` with a matching `fingerprint` applies the plan. `ImportPage.tsx` is the one component that drives all three.
- Import is all-or-nothing: one bad line blocks the file. A partial import would mean fixing three lines and re-importing a file whose other 197 are already in, and nothing in the schema can tell the second copy apart.
- Exact duplicates are counted and reported, not refused (`countImportDuplicates`, one query over the file's date range). Two identical coffees on one day are real.
- There is no server-side state between the three outcomes: the client re-sends the file (and, once known, the mapping) on every request, and the file is sniffed fresh each time, then rewound and planned.
- `Import.Fingerprint` is a digest of what was read; the client echoes it back on confirm, so a file swapped between steps is refused rather than imported unseen.
- Row validation enforces what `POST /api/v1/transactions` enforces (amount, type, note length, future limit). Don't add a laxer way in.
- The two numeric limits live in `server/internal/txnrule` and are read from there by the transaction handlers and the importer alike. Never copy the numbers.
- `GET /api/v1/transactions/export` answers a CSV, but a plain `<a href>` can't carry this request's `Authorization` header -- `lib/api/import.ts`'s `downloadTransactionsExport` fetches it with the access token attached and turns the response into a download itself (an object URL + a synthetic `<a download>`).
