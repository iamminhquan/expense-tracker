package api_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/api"
	"expensetracker/internal/auth"
)

// postImport uploads a CSV the same way handlers_test.postImport does for
// the HTML side, authenticated with an access token instead of a session
// cookie.
func postImport(t *testing.T, deps api.Deps, router http.Handler, userID int64, csv string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	file, err := form.CreateFormFile("file", "upload.csv")
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

	token, _, err := auth.IssueAccessToken(userID, deps.JWTSecret)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/transactions/import", body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// exportHeaderCSV is a file shaped exactly like this app's own export
// (see export_handlers.go's exportColumns), which Sniff recognizes and
// skips the mapping screen for entirely -- the same shortcut
// handlers/import_handlers_test.go's "spend-all.csv" fixture relies on.
func exportHeaderCSV(categoryName string) string {
	return fmt.Sprintf("Date,Type,Category,Amount,Note\n2026-01-15,expense,%s,50000,Groceries run\n", categoryName)
}

func TestImportExactFormatPreviewThenConfirm(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	createTestCategory(t, deps, router, userID, "Groceries", "expense")

	rec := postImport(t, deps, router, userID, exportHeaderCSV("Groceries"), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview POST /api/transactions/import = %d %s, want 200", rec.Code, rec.Body.String())
	}
	preview := decodeJSON[struct {
		Preview     bool   `json:"preview"`
		RowCount    int    `json:"rowCount"`
		Importable  bool   `json:"importable"`
		Fingerprint string `json:"fingerprint"`
	}](t, rec)
	if !preview.Preview || preview.RowCount != 1 || !preview.Importable {
		t.Fatalf("preview = %+v, want {Preview:true RowCount:1 Importable:true ...}", preview)
	}

	rec = postImport(t, deps, router, userID, exportHeaderCSV("Groceries"), map[string]string{
		"confirm": "1", "fingerprint": preview.Fingerprint,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm POST /api/transactions/import = %d %s, want 200", rec.Code, rec.Body.String())
	}
	result := decodeJSON[struct {
		Imported int    `json:"imported"`
		Month    string `json:"month"`
	}](t, rec)
	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1", result.Imported)
	}
	if result.Month != "2026-01" {
		t.Errorf("Month = %q, want %q", result.Month, "2026-01")
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/transactions?month=2026-01", userID))
	list := decodeJSON[struct{ TotalCount int64 }](t, rec)
	if list.TotalCount != 1 {
		t.Errorf("transactions in 2026-01 after import = %d, want 1", list.TotalCount)
	}
}

func TestImportRejectsWrongFingerprint(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	createTestCategory(t, deps, router, userID, "Groceries", "expense")

	rec := postImport(t, deps, router, userID, exportHeaderCSV("Groceries"), map[string]string{
		"confirm": "1", "fingerprint": "not-the-real-fingerprint",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("confirm with wrong fingerprint = %d %s, want 409", rec.Code, rec.Body.String())
	}
}

func TestImportUnknownFormatAsksForMapping(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)

	foreignCSV := "When,What,How much\n2026-01-15,Coffee,-50000\n"
	rec := postImport(t, deps, router, userID, foreignCSV, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/transactions/import (foreign format) = %d %s, want 200", rec.Code, rec.Body.String())
	}
	mapping := decodeJSON[struct {
		NeedsMapping bool     `json:"needsMapping"`
		Columns      []string `json:"columns"`
	}](t, rec)
	if !mapping.NeedsMapping {
		t.Fatal("NeedsMapping = false for an unrecognized CSV format, want true")
	}
	if len(mapping.Columns) != 3 {
		t.Errorf("Columns = %v, want 3 columns", mapping.Columns)
	}
}
