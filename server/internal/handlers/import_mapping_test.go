package handlers_test

import (
	"context"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

// foreignFile is a file no export of this app ever wrote: other column
// names, another order, day-first dates, and the expense/income split
// carried by the sign instead of a column.
const foreignFile = "Group,When,Value,Memo\n" +
	"Food & Drink,25/07/2026,-45000,Cà phê\n" +
	"Salary,01/07/2026,20000000,July\n"

// foreignMapping is what the mapping screen would submit for foreignFile.
var foreignMapping = map[string]string{
	"mapped": "1", "date_col": "1", "amount_col": "2", "category_col": "0",
	"note_col": "3", "type_col": "-1", "date_layout": "dmy-slash",
	"negative_is_expense": "1",
}

func TestImportAsksHowToReadAForeignFile(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-map@example.com", "s3cret-pass")

	body := postImport(t, router, cookie, foreignFile, nil).Body.String()

	if !strings.Contains(body, `id="import-mapping-form"`) {
		t.Fatalf("a foreign file did not get a mapping screen, got: %s", body)
	}
	// The guess has to arrive selected, or the screen is just a form to
	// fill in by hand.
	for _, want := range []string{
		`<option value="1" selected>When</option>`,
		`<option value="2" selected>Value</option>`,
		`<option value="dmy-slash" selected>`,
		`name="negative_is_expense" value="1" checked`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("mapping screen does not pre-select %q", want)
		}
	}
}

func TestImportSkipsTheMappingScreenForItsOwnExport(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-exact@example.com", "s3cret-pass")

	body := postImport(t, router, cookie,
		"Date,Type,Category,Amount,Note\n2026-08-11,expense,Food & Drink,45000,Cà phê\n", nil).Body.String()

	if strings.Contains(body, `id="import-mapping-form"`) {
		t.Error("the app's own export was sent to the mapping screen")
	}
	if !strings.Contains(body, `name="confirm"`) {
		t.Errorf("the app's own export did not go straight to a preview, got: %s", body)
	}
}

func TestImportRefusesAMappingThatCannotWork(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-badmap@example.com", "s3cret-pass")

	cases := []struct {
		name    string
		change  map[string]string
		message string
	}{
		{"one column twice", map[string]string{"amount_col": "1"}, "cannot be both"},
		{"no date column", map[string]string{"date_col": "-1"}, "holds the date"},
		{"no category and no fallback", map[string]string{"category_col": "-1"}, "file everything under"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]string{}
			for k, v := range foreignMapping {
				fields[k] = v
			}
			for k, v := range tc.change {
				fields[k] = v
			}

			body := postImport(t, router, cookie, foreignFile, fields).Body.String()

			if !strings.Contains(body, tc.message) {
				t.Errorf("mapping was accepted or misreported, want %q, got: %s", tc.message, body)
			}
			if !strings.Contains(body, `id="import-mapping-form"`) {
				t.Error("a rejected mapping did not come back to the mapping screen")
			}
		})
	}
}

// TestImportRunsAForeignFileAllTheWayIn is the whole point of the mapping
// step: a file in someone else's shape ends up as this account's rows.
func TestImportRunsAForeignFileAllTheWayIn(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "import-foreign@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")
	user, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	preview := postImport(t, router, cookie, foreignFile, foreignMapping)
	fingerprint := fingerprintFrom(t, preview.Body.String())

	fields := map[string]string{"confirm": "1", "fingerprint": fingerprint}
	for k, v := range foreignMapping {
		fields[k] = v
	}
	rec := postImport(t, router, cookie, foreignFile, fields)

	if redirect := rec.Header().Get("HX-Redirect"); !strings.Contains(redirect, "imported=2") {
		t.Fatalf("HX-Redirect = %q, want 2 imported: %s", redirect, rec.Body.String())
	}
	if n := tableCount(t, deps,
		"SELECT count(*) FROM transactions WHERE user_id = $1 AND type = 'expense' AND amount = 45000 AND occurred_on = '2026-07-25'",
		user.ID); n != 1 {
		t.Errorf("the negative July 25th line did not land as a 45000 expense (%d rows matched)", n)
	}
	if n := tableCount(t, deps,
		"SELECT count(*) FROM transactions t JOIN categories c ON c.id = t.category_id WHERE t.user_id = $1 AND t.type = 'income' AND c.slug = 'salary'",
		user.ID); n != 1 {
		t.Errorf("the positive line did not land as income on the Salary default (%d rows matched)", n)
	}
}

func TestImportPreviewGoesBackToTheMappingItCameFrom(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-back@example.com", "s3cret-pass")

	fields := map[string]string{"back": "1"}
	for k, v := range foreignMapping {
		fields[k] = v
	}
	// A mapping that is not the guess: the note column left out.
	fields["note_col"] = "-1"

	body := postImport(t, router, cookie, foreignFile, fields).Body.String()

	if !strings.Contains(body, `id="import-mapping-form"`) {
		t.Fatalf("Back did not return to the mapping screen, got: %s", body)
	}
	if !strings.Contains(body, `<option value="-1" selected>— none —</option>`) {
		t.Error("Back lost the mapping that was submitted and fell back to the guess")
	}
}

// TestImportPointsAtTheDateFormatWhenMostLinesFailOnIt covers the one wrong
// answer on the mapping screen that produces plausible-looking failures
// everywhere else: pick month-first for day-first dates and every line past
// the 12th breaks for what reads like its own reason.
func TestImportPointsAtTheDateFormatWhenMostLinesFailOnIt(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-datehint@example.com", "s3cret-pass")

	fields := map[string]string{}
	for k, v := range foreignMapping {
		fields[k] = v
	}
	fields["date_layout"] = "mdy-slash"

	body := postImport(t, router, cookie, foreignFile, fields).Body.String()

	if !strings.Contains(body, "date format on the mapping screen is probably wrong") {
		t.Errorf("preview does not point at the date format, got: %s", body)
	}
}
