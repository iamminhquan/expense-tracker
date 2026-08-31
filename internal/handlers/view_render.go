package handlers

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"time"

	"expensetracker/internal/auth"
	"expensetracker/internal/csrf"
	"expensetracker/internal/format"
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

// viewData is what every rendered page can count on carrying, whatever else
// it holds: the CSRF token its forms echo back, the theme the <html>
// element is stamped with, and -- on an authenticated page -- the fields
// layout.html's two nav bars read.
//
// A page's own data embeds it instead of restating the fields.
// html/template promotes an embedded struct's fields exactly the way Go
// does, so {{.CSRFToken}} and {{.ShowNav}} reach these from any page
// without the template knowing they are nested.
//
// ShowNav has no explicit false anywhere: a pre-auth page leaves it zero,
// which is what layout.html's {{if .ShowNav}} blocks read.
type viewData struct {
	CSRFToken string
	Theme     string

	ShowNav       bool
	ActiveNav     string
	UserName      string
	UserInitial   string
	Greeting      string
	HeaderBalance balanceSummary
	EmailVerified bool
}

// pageView is any page's own data. It is satisfied by embedding viewData
// and passing a pointer, which is what lets render fill the shared fields
// in without knowing anything else about the page.
type pageView interface {
	view() *viewData
}

func (v *viewData) view() *viewData { return v }

// render executes page's "layout" template. It always injects CSRFToken; if
// active is non-empty it also injects the ShowNav/ActiveNav/UserName/
// UserInitial/HeaderBalance/EmailVerified fields layout.html's nav blocks
// need, by loading the authenticated user via deps.Queries. Pre-auth pages
// (login/register) pass active="" so nav data is skipped entirely and
// layout.html's {{if .ShowNav}} blocks correctly stay hidden -- a missing
// map key evaluates falsy in html/template's {{if}}, so no explicit false
// is needed.
func render(w http.ResponseWriter, r *http.Request, deps Deps, page string, active string, data map[string]any) {
	renderNamed(w, r, deps, page, "layout", active, data)
}

// renderNamed is render's more general form: it executes tmplName instead
// of always "layout", for fragment responses (a swapped-in row, a
// tab-switch card body, etc.) that must not re-render the full page shell.
func renderNamed(w http.ResponseWriter, r *http.Request, deps Deps, page string, tmplName string, active string, data map[string]any) {
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
			serverError(w, "render: load nav data", err)
			return
		}
		for k, v := range navData {
			data[k] = v
		}
	}

	renderFragment(w, r, deps, page, tmplName, data)
}

// renderView is render's typed form: the same full-page render, for a page
// whose data is its own struct rather than a map.
func renderView(w http.ResponseWriter, r *http.Request, deps Deps, page string, active string, data pageView) {
	renderViewNamed(w, r, deps, page, "layout", active, data)
}

// renderViewNamed fills in the fields every template shares -- the CSRF
// token, the theme, and the nav data when active is non-empty -- and
// renders tmplName against data.
//
// It is renderNamed with a struct in place of the map. The map version
// still serves the pages that have not been converted; when the last one
// has, these two names take over.
func renderViewNamed(w http.ResponseWriter, r *http.Request, deps Deps, page string, tmplName string, active string, data pageView) {
	v := data.view()
	v.CSRFToken = csrf.TokenFromRequest(r)
	// Every page carries a theme, including the pre-auth ones with no user
	// to read a preference from. authPageView overwrites this with the
	// stored preference for authenticated pages.
	v.Theme = defaultTheme

	if active != "" {
		nav, err := authPageView(r, deps, active)
		if err != nil {
			serverError(w, "render: load nav data", err)
			return
		}
		nav.CSRFToken = v.CSRFToken
		*v = nav
	}

	renderFragment(w, r, deps, page, tmplName, data)
}

// renderFragment executes tmplName against data exactly as it stands: no
// CSRF token, no theme, no nav fields are added to it.
//
// That is what lets a fragment hand its template a typed struct rather than
// the map renderNamed has to be able to write those fields into -- and a
// struct is what turns a field the response forgot into a compile error
// instead of a silently empty column. Use it for the fragments whose
// templates need nothing from the page shell: a swapped-in row, an inline
// edit form, a confirm prompt.
//
// A fragment whose template does reach for {{.CSRFToken}} -- the settings
// and import forms do, since they post as plain forms rather than through
// htmx's header -- belongs on renderNamed instead.
func renderFragment(w http.ResponseWriter, r *http.Request, deps Deps, page, tmplName string, data any) {
	tmpl, ok := deps.Templates[page]
	if !ok {
		log.Printf("render: no template registered for page %q", page)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Buffered rather than written straight to w: a template that fails
	// halfway has already written a broken page otherwise, and the 500
	// below could not be sent at all.
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

// currentHeaderBalance builds the balance the nav widget shows. It is always
// the real current month, never the month a page happens to be browsing --
// the widget sits in the layout, above and outside the month picker.
//
// Mutation handlers need this as well as full page renders: their responses
// swap the widget back out of band, and without a fresh figure the number a
// user is looking at goes wrong the moment they add anything.
func currentHeaderBalance(r *http.Request, deps Deps, userID int64) (balanceSummary, error) {
	from, to := currentMonthRange()
	totals, err := deps.Queries.MonthlyTotals(r.Context(), sqlcgen.MonthlyTotalsParams{
		UserID: userID, OccurredOn: from, OccurredOn_2: to,
	})
	if err != nil {
		return balanceSummary{}, err
	}
	return newBalanceSummary(totals.TotalExpense, totals.TotalIncome, totals.CarriedOver), nil
}

// authPageView loads the fields every authenticated page's nav needs: the
// user's username/initial (the nav shows the handle, not the free-text
// name collected at signup), which nav link is active, and the real
// current month's balance for the header widget (independent of whatever
// month the page itself is browsing).
//
// The dashboard's greeting is built here too, for two reasons: this is the
// only place that loads the username it addresses, and it runs on the
// fragment render as well as the full one -- built anywhere else, the
// greeting would vanish the first time a month switch swapped the section
// it sits in.
func authPageView(r *http.Request, deps Deps, active string) (viewData, error) {
	userID, _ := auth.UserIDFromContext(r.Context())
	user, err := deps.Queries.GetUserByID(r.Context(), userID)
	if err != nil {
		return viewData{}, err
	}
	initial := "?"
	if runes := []rune(user.Username); len(runes) > 0 {
		initial = strings.ToUpper(string(runes[0]))
	}

	headerBalance, err := currentHeaderBalance(r, deps, userID)
	if err != nil {
		return viewData{}, err
	}

	// Vietnam time, deliberately fixed rather than read from the client: the
	// app is single-country (as every month boundary here already assumes),
	// so a server clock in another zone is the only skew worth correcting,
	// and correcting it needs no cookie and no load-time JavaScript.
	return viewData{
		ShowNav:       true,
		ActiveNav:     active,
		UserName:      user.Username,
		UserInitial:   initial,
		Greeting:      format.GreetingLine(time.Now().In(vietnamLocation), user.Username),
		Theme:         user.Theme,
		HeaderBalance: headerBalance,
		EmailVerified: user.EmailVerified,
	}, nil
}

// authPageData is authPageView spread back into the map the pages that
// still render from one expect. It goes when the last of them is converted.
func authPageData(r *http.Request, deps Deps, active string) (map[string]any, error) {
	v, err := authPageView(r, deps, active)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ShowNav":       v.ShowNav,
		"ActiveNav":     v.ActiveNav,
		"UserName":      v.UserName,
		"UserInitial":   v.UserInitial,
		"Greeting":      v.Greeting,
		"Theme":         v.Theme,
		"HeaderBalance": v.HeaderBalance,
		"EmailVerified": v.EmailVerified,
	}, nil
}

// serverError is the answer to a failure that is ours rather than the
// caller's: it records what was being attempted and answers a bare 500.
//
// The client message is deliberately fixed and says nothing. Which query
// failed is a fact about our schema, and the person who can act on it is
// reading the log, not the page -- so what varies between call sites is the
// line in the log, and only that.
func serverError(w http.ResponseWriter, attempted string, err error) {
	log.Printf("%s: %v", attempted, err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}
