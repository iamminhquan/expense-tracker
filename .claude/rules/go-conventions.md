---
paths:
  - "**/*.go"
---

# Go coding conventions


Follow Google's official Go guidance, in its own stated order of priority: **clarity > simplicity > concision > maintainability > consistency**. The sources, in precedence order, are the [Google Go Style Guide](https://google.github.io/styleguide/go/) (Style Guide → Style Decisions → Best Practices), [Effective Go](https://go.dev/doc/effective_go), and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Where this repo has already settled a question, match the surrounding code — a consistent codebase beats a locally-optimal snippet.

**Formatting and tooling.** `gofmt` output is the only accepted formatting; `gofmt -l .` must print nothing and `go vet ./...` must be clean before a commit. There is no line-length limit, but prefer breaking a long expression over letting one line carry three ideas. No linter config is checked in; don't add one without asking.

**Naming.**
- `MixedCaps` / `mixedCaps`, never `snake_case` or `SCREAMING_CASE`. Initialisms keep one case throughout: `userID`, `accessToken`, `URLParam`, not `userId`, `accessTOKEN` or `UrlParam`.
- Name length scales with scope. A loop or a three-line closure gets `f`, `p`, `m`, `row`; a package-level identifier gets a name that reads on its own. `deps`, `w`, `r`, `ctx` are house names — use them.
- Package names are short, lowercase, single-word, no underscores, no plurals where a singular reads better, and never `util`, `common`, `helpers`, or `base`. The name is part of every reference, so avoid stutter: `config.Load`, not `config.LoadConfig`.
- No `Get` prefix on accessors. `TokenFromRequest`, not `GetToken`. (The sqlc-generated `GetUserByID` is generated from the query name and is exempt.)
- Receivers are one or two letters, consistent across every method on the type: `(f txnFilters)`, `(p pager)`, `(s txnScope)`.

**Errors.**
- Error strings are lowercase and unpunctuated: `"session expired"`, not `"Session expired."`.
- Wrap with `%w` when the caller might reasonably inspect the cause, `%v` when it is just context; always add what was being attempted: `fmt.Errorf("read header: %w", err)`.
- Compare with `errors.Is` and `errors.As`, never `==` or a bare type assertion. The Postgres unique-violation check is the pattern here: `errors.As(err, &pgErr) && pgErr.Code == "23505"`.
- Handle every error. A deliberate discard needs `_ =` and, unless the reason is obvious, a comment. `userID, _ := auth.UserIDFromContext(...)` inside a `RequireAuth` group is the one routine exception — the middleware guarantees the value.
- Sentinel errors are `Err`-prefixed package-level `var`s (`config.ErrMissingDatabaseURL`).
- Don't `panic` in normal flow outside a build-time invariant that can't be worked around any other way.

**Control flow.** Keep the happy path at the left margin: handle the error, return early, and leave the success case unindented. Prefer a `switch` to a chain of `else if`. Avoid `else` after a `return`.

**Functions and types.**
- `ctx context.Context` is the first parameter, never a struct field.
- Return concrete types; accept interfaces only where more than one implementation is real. This codebase has almost none by design.
- Named result parameters are for documenting what a multi-value return means (`monthRangeFor`'s `(from, to pgtype.Date)`), not for naked returns in long functions — a function you have to scroll to read should return its values explicitly.
- Avoid `any` unless a function genuinely has to accept more than one concrete type and there's no shared interface to ask for instead.
- Declare an empty slice as `var s []string` unless nil and empty differ to the caller — and when they do, say so, as `i18n.SlugsMatching` does. **Exception, and it matters more than this rule generally does:** a slice that ends up in a JSON response must never be a bare `var` that might stay nil -- see `.claude/rules/json-api-conventions.md`'s null-vs-`[]` rule, which is this same Go behavior causing a real bug once a response crosses into JSON.
- Prefer a small struct with a constructor over a long parameter list once a value has behaviour of its own (`newPager`, `newBalanceDTO`, `txnFilters`).

**Comments.** Every exported identifier gets a doc comment starting with its own name, in complete sentences. Beyond that, comment the *why*, not the what — a constraint, a workaround, a decision someone would otherwise undo. That is the prevailing style here and the repo depends on it: the reason the refresh cookie's `SameSite` tracks `SecureCookies`, the reason `::bigint` casts the nullable filter params, the reason month bounds are Vietnam-local. Don't narrate code that already reads clearly.

**Package layout.** Everything lives under `internal/`, one package per responsibility, no import cycles. New shared helpers go in the package that owns the concept, not in a grab-bag: `internal/format` is where display strings go, `internal/pgval` the pgtype wrappers, `internal/txnrule` the limits a transaction has to satisfy whichever way it arrives, `internal/i18n` the default-category name table. A helper that needs neither a request nor a database belongs in one of those, or gets its own package, rather than growing a file in `internal/api` that only happens to live there.

**Tests.** Standard library `testing` only — no testify, no assert helpers (`stretchr/testify` in `go.sum` is a transitive dependency of golang-migrate, not ours to use).
- Default to the external test package `package foo_test`, which forces the test through the public API. When a test genuinely needs unexported identifiers, use `package foo` and name the file `*_internal_test.go`.
- Table-driven where the cases are uniform; subtests via `t.Run` when a failure needs a name.
- Failure messages read `FuncName(input) = got, want want`, with `%q` for strings: `t.Errorf("vnd(%d) = %q, want %q", tc.in, got, tc.want)`.
- `t.Fatalf` when the test cannot continue, `t.Errorf` when it can.
- Setup helpers call `t.Helper()` first, and clean up with `t.Cleanup`.
- DB-touching tests read `TEST_DATABASE_URL` and `t.Skip` when it is unset, so `go test ./...` still passes on a machine with no Postgres.
- `go test` passing is not the same as a change actually working once it crosses into JSON or a browser -- `httptest` doesn't reproduce a real JSON encoder's null-vs-`[]` distinction or a real browser's cookie policy. See `.claude/rules/json-api-conventions.md`'s "verify with a real browser" rule before trusting a green test suite alone on a response-shape or cookie change.

