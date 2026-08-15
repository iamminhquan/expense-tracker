package handlers

// The three values users.theme accepts (mirrored by that column's CHECK
// constraint in migration 000007). "auto" is not resolved server-side: it
// renders as class="auto" and lets layout.html's prefers-color-scheme media
// query pick light or dark, so the server never has to guess what the
// visitor's operating system is set to.
const (
	themeAuto  = "auto"
	themeLight = "light"
	themeDark  = "dark"

	// defaultTheme is what pre-auth pages (login, register) render with,
	// since they have no user whose preference could be loaded.
	defaultTheme = themeAuto
)

// validTheme reports whether v is one of the accepted preferences. Checking
// in Go keeps a bad request a 400 rather than letting the CHECK constraint
// turn it into a 500.
func validTheme(v string) bool {
	return v == themeAuto || v == themeLight || v == themeDark
}
