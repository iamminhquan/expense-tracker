---
paths:
  - "internal/inbound/**"
  - "internal/inboxproc/**"
  - "internal/bankmail/**"
  - "internal/classify/**"
  - "internal/handlers/inbox_webhook.go"
  - "emailworker/**"
  - "internal/handlers/settings_inbox.go"
  - "internal/web/templates/settings_inbox.html"
  - "internal/database/queries/bank_emails.sql"
  - "internal/database/queries/bank_accounts.sql"
  - "internal/database/queries/category_hints.sql"
---

# Email ingestion

Bank emails forwarded to `<token>@in.<domain>` become transactions without the owner typing anything.

**Flow:** the Cloudflare Email Worker (`emailworker/`) signs the JSON body with HMAC-SHA256 and POSTs it to `/inbox/{token}` → `inbox_webhook.go` stores the raw message and answers 200 → a goroutine in `internal/inboxproc` reads each `pending` row → `internal/bankmail` parses it into a `Notice` → a category is chosen → the transaction is created.

## Receiving

- Change anything in `internal/inbound` (payload field names, HMAC, inbox token, body cap) and `emailworker/src/index.js` in the **same commit**, then deploy the Worker (`npx wrangler deploy` from `emailworker/`). A mismatch silently stops email from arriving.
- Keep `/inbox/{token}` public and exempt from CSRF — its caller is the Worker, not a browser.
- Three checks decide trust: the token names an account, the signature proves the Worker sent it, the sender domain proves a bank sent it. A message that fails only the sender check is stored as `ignored`, never dropped.
- The handler stores first and answers 200 without reading. Never parse in the request path, so a wrong parser can't lose an email.

## Parsing

- `internal/bankmail` takes `(from, subject, body)` and returns a `Notice`. It touches no database and no network.
- It fails closed. It reads MB's transfer-confirmation template, debit side only. Don't add a guessed parser for TPBank or incoming credit; an unrecognised shape becomes a `failed` row. `parseMBAmount` validates the amount's shape instead of tolerating variants.
- Use `bankmail.NoteKey` for every note lookup. It is exported because processing an email and correcting a category must share one rule.

## Processing

- `internal/inboxproc` is the only writer. It runs on `context.Background()` and claims a row with `UPDATE ... WHERE status='pending' RETURNING`, so two goroutines never process one message.
- Close each row as `imported`, `ignored` or `failed`. `ignored` means nothing is wrong (unknown sender, marketing mail, self-transfer); `failed` means our own bug. Don't mix them, or the `failed` list becomes noise.

## Choosing the category

- `category_hints` (keyed by `NoteKey(description)`) decides first; a miss falls back to the `other` / `other_income` **slug** (see `categories.md`).
- A hint decides where a transaction lands, never whether it exists. If the hint lookup errors, fall through to the fallback and still create the row.
- Only rows with `source='email'` teach a hint, when the owner corrects a row marked `auto`. Ignore a hint that points at a category of the wrong type or another account's. `category_hints.category_id` cascades on delete, unlike `transactions.category_id`.
- On a miss, `internal/classify` asks Gemini to pick one id from that account's own categories, already filtered to the notice's direction, then re-checks the id in Go and writes the answer back as a hint. Any failure (no key, 429, 500, timeout, bad answer, id outside the list) falls back to `Other` and the transaction is still created.
- `GEMINI_API_KEY` and `GEMINI_MODEL` are optional. The request is plain `net/http` with the key in the `x-goog-api-key` header. Read the comments in `classify.go` before changing the request: its schema and token-budget details are the API's quirks and each is pinned by a test.

## Self-transfers

- A notice whose beneficiary account is already in `bank_accounts` is closed as `ignored` with a reason, not recorded.
- `bank_accounts` learns only from proof: the debit account (`Tài khoản trích nợ`) of a notice delivered to that user's own inbox. Never write the beneficiary side, and never accept a matching account-holder name as proof.
- Every failure on this path (unparsed account, lookup error, account not yet learned) answers "not internal" and records the transaction.

## Why

- Parsers fail closed because a guess puts invented money in a real ledger; an unrecognised email stays visible and replayable.
- Self-transfers err toward recording because a wrongly recorded row is visible and deletable, while a wrongly ignored payment is missing money the owner can't know about.
