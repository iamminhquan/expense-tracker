package handlers

import (
	"bytes"
	"log"
	"net/http"
	"strings"

	"expensetracker/internal/auth"
	"expensetracker/internal/csrf"
	"expensetracker/internal/sqlcgen"
)

// isFragmentRequest reports whether r is an explicit htmx fragment request
// (e.g. the month-dropdown's hx-get) rather than a boosted top-level nav
// click. Both carry HX-Request: true, but only the boosted click also
// carries HX-Boosted: true, since hx-boost applies to the plain nav <a>
// itself while the month dropdown declares its own hx-get/hx-target -- so a
// handler that special-cases HX-Request alone would wrongly hand a boosted
// nav click just the fragment instead of the full page shell.
func isFragmentRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Boosted") != "true"
}

// render executes page's "layout" template. It always injects CSRFToken; if
// active is non-empty it also injects the ShowNav/ActiveNav/UserName/
// UserInitial/HeaderBalance fields layout.html's nav blocks need, by loading
// the authenticated user via deps.Queries. Pre-auth pages (login/register) pass
// active="" so nav data is skipped entirely and layout.html's
// {{if .ShowNav}} blocks correctly stay hidden -- a missing map key
// evaluates falsy in html/template's {{if}}, so no explicit false is
// needed.
func render(w http.ResponseWriter, r *http.Request, deps Deps, page string, active string, data map[string]any) {
	renderNamed(w, r, deps, page, "layout", active, data)
}

// renderNamed is render's more general form: it executes tmplName instead
// of always "layout", for fragment responses (a swapped-in row, a
// tab-switch card body, etc.) that must not re-render the full page shell.
func renderNamed(w http.ResponseWriter, r *http.Request, deps Deps, page string, tmplName string, active string, data map[string]any) {
	tmpl, ok := deps.Templates[page]
	if !ok {
		log.Printf("render: no template registered for page %q", page)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if data == nil {
		data = map[string]any{}
	}
	data["CSRFToken"] = csrf.TokenFromRequest(r)
	// Every page must carry a theme, including the pre-auth ones that have
	// no user to read a preference from: html/template renders a missing map
	// key as the literal "<no value>", which would otherwise land inside
	// layout.html's class attribute. authPageData overwrites this with the
	// stored preference for authenticated pages.
	data["Theme"] = defaultTheme

	if active != "" {
		navData, err := authPageData(r, deps, active)
		if err != nil {
			log.Printf("render: load nav data: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		for k, v := range navData {
			data[k] = v
		}
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		log.Printf("render: execute template %q (block %q): %v", page, tmplName, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("render: write response for %q: %v", page, err)
	}
}

// authPageData loads the fields every authenticated page's nav needs: the
// user's display name/initial, which nav link is active, and the real
// current month's balance for the header widget (independent of whatever
// month the page itself is browsing).
func authPageData(r *http.Request, deps Deps, active string) (map[string]any, error) {
	userID, _ := auth.UserIDFromContext(r.Context())
	user, err := deps.Queries.GetUserByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	initial := "?"
	if runes := []rune(user.Name); len(runes) > 0 {
		initial = strings.ToUpper(string(runes[0]))
	}

	from, to := currentMonthRange()
	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{UserID: userID, OccurredOn: from, OccurredOn_2: to})
	if err != nil {
		return nil, err
	}
	headerBalance := newBalanceCard(totals.TotalExpense, totals.TotalIncome, totals.CarriedOver, from.Time, "header")

	return map[string]any{
		"ShowNav":       true,
		"ActiveNav":     active,
		"UserName":      user.Name,
		"UserInitial":   initial,
		"Theme":         user.Theme,
		"HeaderBalance": headerBalance,
	}, nil
}
