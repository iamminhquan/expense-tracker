---
paths:
  - "internal/web/**"
  - "internal/handlers/view_*.go"
---

# Templates and static assets

`internal/web`: templates and static assets are both `go:embed`ed into the binary.

## Templates

- `web.Templates(funcs)` is the single place page sets are built. Each page (`auth`, `categories`, `transactions`, `dashboard`, `settings`) is its own `*template.Template` parsing the shared partials plus that page's files, so only one `{{define "content"}}` is ever in scope. `cmd/server` and the handler tests both call it; never hand-roll a `ParseFiles` list.
- Shared partials, parsed into every page set: `layout.html` (the shell), `nav.html` (desktop bar, mobile bottom bar, wordmark), `mobile_header.html` (the two sticky mobile tiers), `month_picker.html` (the one month control, rendered by both breakpoints on both month-scoped pages), `user_menu.html`, `header_balance.html`.
- Add a page-specific file to `pageTemplates` in `internal/web/web.go`.

## Static assets

- CSS and JS live in `internal/web/static/` and are served at `/static/` by `web.StaticHandler()`: a public route, ETag'd, `Cache-Control: no-cache`, because `embed.FS` has no ModTime to revalidate against.
- **Never put a `<style>` or an inline `<script>` in a template.**
- `app.css` and `app.js` load from `<head>`. Everything in `app.js` is a delegated listener on `document`, because `hx-boost` replaces only `<body>` and a head script runs once per full page load.
- A page-specific script (`charts.js`, `categories.js`) ships as a `<script src>` *inside* that page's swapped content. htmx re-executes it on swap, which is what rebuilds the dashboard charts on a month switch.

## Rendering (`internal/handlers/view_render.go`)

- `render(w, r, deps, page, active, data)`: the full page. It executes the `"layout"` block and, if `active != ""`, loads the current user and fills the nav data (`ShowNav`, `ActiveNav`, `UserName`, `UserInitial`, `Greeting`, `Theme`, `HeaderBalance`, `EmailVerified`).
- `renderNamed(w, r, deps, page, tmplName, active, data)`: a named sub-template instead of the layout, for htmx fragments that still need the shell's shared fields (a tab body, a month section).
- `renderFragment(w, r, deps, page, tmplName, data)`: a named sub-template against `data` exactly as given, adding nothing. For fragments that need nothing from the shell: a swapped-in row, an inline edit form, a confirm prompt.
- Always pass a struct, never a map. A page's data type embeds `viewData` (CSRF token, theme, nav fields) and satisfies `pageView` by being passed as a pointer. `html/template` promotes the embedded fields, so `{{.CSRFToken}}` works unchanged. Fragments served by `renderFragment` embed nothing.
- Why a struct: a template asking for a key its map lacks prints nothing and reports no error, so a forgotten field shipped a blank column or an empty form unnoticed. A struct makes it a compile error.
- Use `isFragmentRequest` to tell a real fragment request (`HX-Request` alone) from a boosted nav click (`HX-Request` + `HX-Boosted`). A handler that branches on `HX-Request` alone hands a boosted click a fragment instead of the page shell.

## Formatting

- Money and date helpers (`VND`, `VNDSigned`, `VNDBalance`, `DateShort`, `DateLong`, `Timestamp`, `CountOf`) live in `internal/format`. They take finished values and return strings, with no request, no `Deps`, no database, so every money and date rule is testable alone.
- `view_funcs.go` maps them, plus `catName` (`i18n.CategoryName`) and `swatches`, onto the names templates call via `handlers.TemplateFuncs()`. Keep that mapping with the templates.
- `format.Timestamp` takes a `*time.Location` rather than reaching for the app's own.
- A transaction row doesn't call the date helpers itself. The list wraps its rows in `txnRow`, whose `Date` method picks the format, and the three handlers that answer with one row call `rowDate` to make the same choice. A page-level "these rows need their year" flag isn't visible from inside `{{range .Transactions}}`.
- The rules are commas for thousands, a trailing ₫ and a spelled-out month (`11 Aug 2026`). The app was first specified in the Vietnamese convention (dots, `dd/mm/yyyy`), which is why the helpers exist instead of templates formatting inline.
