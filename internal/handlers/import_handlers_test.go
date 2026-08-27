package handlers_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
)

// postImport uploads a CSV the way the browser does: a multipart form,
// posted over htmx so the answer is a fragment rather than a page.
func postImport(t *testing.T, router http.Handler, cookie *http.Cookie, csv string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	file, err := form.CreateFormFile("file", "spend-all.csv")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := file.Write([]byte(csv)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	for name, value := range fields {
		if err := form.WriteField(name, value); err != nil {
			t.Fatalf("write field %s: %v", name, err)
		}
	}
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}

	tok := csrfTokenFor(t, router)
	req := httptest.NewRequest(http.MethodPost, "/transactions/import", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("HX-Request", "true")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestImportPageOffersAnUploadForm(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-page@example.com", "s3cret-pass")

	req := httptest.NewRequest(http.MethodGet, "/transactions/import", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /transactions/import = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`type="file"`, `hx-post="/transactions/import"`, `hx-encoding="multipart/form-data"`} {
		if !strings.Contains(body, want) {
			t.Errorf("import page does not contain %q", want)
		}
	}
}

// TestImportPreviewDescribesTheFileWithoutWritingAnything is the whole point
// of the two-step flow: nothing reaches the database until the second post.
func TestImportPreviewDescribesTheFileWithoutWritingAnything(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "import-preview@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")
	user, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	rec := postImport(t, router, cookie, "Date,Type,Category,Amount,Note\n"+
		"2026-08-11,expense,Food & Drink,45000,Cà phê\n"+
		"2026-08-12,expense,Cà phê,50000,\n", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("preview POST = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Cà phê") {
		t.Errorf("preview does not name the category it would create, got: %s", body)
	}
	if !strings.Contains(body, `name="fingerprint"`) {
		t.Error("preview carries no fingerprint field for the confirm step")
	}
	if n := tableCount(t, deps, "SELECT count(*) FROM transactions WHERE user_id = $1", user.ID); n != 0 {
		t.Errorf("preview wrote %d transactions, want 0", n)
	}
	if n := tableCount(t, deps, "SELECT count(*) FROM categories WHERE user_id = $1", user.ID); n != 0 {
		t.Errorf("preview created %d categories, want 0", n)
	}
}

func tableCount(t *testing.T, deps handlers.Deps, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := deps.DB.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return n
}

// fingerprintFrom digs the confirm step's token out of a rendered preview,
// which is the only place a test can get it -- it is a digest of the file
// the server just read.
func fingerprintFrom(t *testing.T, body string) string {
	t.Helper()
	match := regexp.MustCompile(`name="fingerprint" value="([0-9a-f]+)"`).FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("preview offers no fingerprint to confirm with: %s", body)
	}
	return match[1]
}

func TestImportConfirmWritesTheRowsAndTheCategoriesTheyNeed(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "import-confirm@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")
	user, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	file := "Date,Type,Category,Amount,Note\n" +
		"2026-08-11,expense,Food & Drink,45000,Cà phê\n" +
		"2026-08-12,expense,Cà phê,50000,\n"

	preview := postImport(t, router, cookie, file, nil)
	fingerprint := fingerprintFrom(t, preview.Body.String())

	rec := postImport(t, router, cookie, file, map[string]string{
		"confirm": "1", "fingerprint": fingerprint,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("confirm POST = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if redirect := rec.Header().Get("HX-Redirect"); !strings.Contains(redirect, "imported=2") {
		t.Errorf("HX-Redirect = %q, want it to report 2 imported", redirect)
	}
	if n := tableCount(t, deps, "SELECT count(*) FROM transactions WHERE user_id = $1", user.ID); n != 2 {
		t.Errorf("imported %d transactions, want 2", n)
	}
	// "Food & Drink" is a shared default, so only the unknown name becomes a
	// category of the account's own.
	if n := tableCount(t, deps, "SELECT count(*) FROM categories WHERE user_id = $1", user.ID); n != 1 {
		t.Errorf("created %d categories, want 1", n)
	}
	if n := tableCount(t, deps,
		"SELECT count(*) FROM transactions t JOIN categories c ON c.id = t.category_id WHERE t.user_id = $1 AND c.slug = 'food_drink'",
		user.ID); n != 1 {
		t.Errorf("%d rows landed on the Food & Drink default, want 1", n)
	}
}

// TestImportRefusesToConfirmAFileItDidNotPreview closes the gap the two-step
// flow opens: the file is re-uploaded, so it could be a different one.
func TestImportRefusesToConfirmAFileItDidNotPreview(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "import-swap@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")
	user, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	previewed := "Date,Type,Category,Amount,Note\n2026-08-11,expense,Food & Drink,45000,\n"
	swapped := "Date,Type,Category,Amount,Note\n2026-08-11,expense,Food & Drink,9999999,\n"
	fingerprint := fingerprintFrom(t, postImport(t, router, cookie, previewed, nil).Body.String())

	rec := postImport(t, router, cookie, swapped, map[string]string{
		"confirm": "1", "fingerprint": fingerprint,
	})

	if !strings.Contains(rec.Body.String(), "not the file you previewed") {
		t.Errorf("swapped file was not refused, got: %s", rec.Body.String())
	}
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("swapped file was imported, want it refused")
	}
	if n := tableCount(t, deps, "SELECT count(*) FROM transactions WHERE user_id = $1", user.ID); n != 0 {
		t.Errorf("swapped file wrote %d transactions, want 0", n)
	}
}

func TestImportPreviewWithABadLineOffersNoImportButton(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-badline@example.com", "s3cret-pass")

	rec := postImport(t, router, cookie, "Date,Type,Category,Amount,Note\n"+
		"2026-08-11,expense,Food & Drink,45000,\n"+
		"11/08/2026,expense,Food & Drink,45000,\n", nil)

	body := rec.Body.String()
	if !strings.Contains(body, "Line 3") {
		t.Errorf("preview does not name the bad line, got: %s", body)
	}
	if strings.Contains(body, `name="fingerprint"`) {
		t.Error("preview offers an import button for a file with a bad line")
	}
}

// TestImportWarnsWhenTheSameFileIsImportedTwice is the answer to the one
// thing all-or-nothing does not prevent: nothing about the schema stops the
// same file going in a second time, so the count is shown and the decision
// left to the person importing.
func TestImportWarnsWhenTheSameFileIsImportedTwice(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	ctx := context.Background()
	email := "import-twice@example.com"
	cookie := loginAndGetCookie(t, router, deps, email, "s3cret-pass")
	user, err := deps.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	t.Cleanup(func() {
		deps.DB.Exec(ctx, "DELETE FROM transactions WHERE user_id = $1", user.ID)
	})

	file := "Date,Type,Category,Amount,Note\n2026-08-11,expense,Food & Drink,45000,Cà phê\n"
	fingerprint := fingerprintFrom(t, postImport(t, router, cookie, file, nil).Body.String())
	if rec := postImport(t, router, cookie, file, map[string]string{"confirm": "1", "fingerprint": fingerprint}); rec.Header().Get("HX-Redirect") == "" {
		t.Fatalf("first import did not go through: %s", rec.Body.String())
	}

	again := postImport(t, router, cookie, file, nil).Body.String()

	if !strings.Contains(again, "already") {
		t.Errorf("second preview does not warn about the duplicate, got: %s", again)
	}
	if !strings.Contains(again, `name="fingerprint"`) {
		t.Error("second preview refuses the import outright, want a warning that can still be confirmed")
	}
}

func TestTransactionsPageLinksToImportAndReportsAFinishedOne(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "import-entry@example.com", "s3cret-pass")

	get := func(path string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, rec.Code)
		}
		return rec.Body.String()
	}

	if body := get("/transactions"); !strings.Contains(body, `href="/transactions/import"`) {
		t.Error("the transactions page offers no way into the importer")
	}
	if body := get("/transactions?imported=2"); !strings.Contains(body, "Imported 2 transactions") {
		t.Errorf("the list does not confirm a finished import, got: %s", body)
	}
	if body := get("/transactions"); strings.Contains(body, "Imported") {
		t.Error("the list confirms an import that never happened")
	}
}
