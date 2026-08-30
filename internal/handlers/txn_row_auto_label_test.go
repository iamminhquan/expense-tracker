package handlers_test

import (
	"html/template"
	"strings"
	"testing"

	"expensetracker/internal/handlers"

	"github.com/jackc/pgx/v5/pgtype"
)

// renderTransactionRow executes transaction_row.html's "transaction_row"
// block on its own, the same way renderMobilePageHeader in
// view_mobile_header_test.go exercises mobile_header.html: a plain map
// stands in for txnRow, since txnRow itself is unexported and its own
// package (handlers, in txn_duplicate_internal_test.go) already covers how
// IsDuplicate gets set. What is under test here is the markup a given
// Source/IsDuplicate combination produces.
func renderTransactionRow(t *testing.T, data map[string]any) string {
	t.Helper()
	tmpl := template.Must(template.New("transaction_row.html").
		Funcs(handlers.TemplateFuncs()).
		ParseFiles("../web/templates/transaction_row.html"))
	var sb strings.Builder
	if err := tmpl.ExecuteTemplate(&sb, "transaction_row", data); err != nil {
		t.Fatalf("execute transaction_row: %v", err)
	}
	return sb.String()
}

func baseRowData() map[string]any {
	return map[string]any{
		"ID": int64(1), "CategorySlug": pgtype.Text{String: "food", Valid: true}, "CategoryName": "", "CategoryColor": "#B45D3A",
		"Description": "Ca phe", "Amount": int64(20000), "Type": "expense",
		"Date": "30 Aug",
	}
}

// TestTransactionRowShowsAutoLabelForEmailSource is the case the label
// exists for: a row the bank-email worker created (source='email') has to
// be visually distinguishable from one the owner typed, so they can tell at
// a glance which entries they never touched.
func TestTransactionRowShowsAutoLabelForEmailSource(t *testing.T) {
	data := baseRowData()
	data["Source"] = "email"
	out := renderTransactionRow(t, data)
	if !strings.Contains(out, ">auto<") {
		t.Errorf("renderTransactionRow(Source=%q) = %q, want it to contain the %q label", "email", out, "auto")
	}
}

// TestTransactionRowCarriesNoAutoLabelForManualSource is the required
// negative case from the plan: a hand-entered row (source='manual') must not
// carry the "auto" label, or the label stops meaning anything.
func TestTransactionRowCarriesNoAutoLabelForManualSource(t *testing.T) {
	data := baseRowData()
	data["Source"] = "manual"
	out := renderTransactionRow(t, data)
	if strings.Contains(out, ">auto<") {
		t.Errorf("renderTransactionRow(Source=%q) = %q, want no %q label", "manual", out, "auto")
	}
}

// TestTransactionRowShowsPossibleDuplicateWhenFlagged checks the row
// template reads IsDuplicate and prints the English copy the plan asks for.
func TestTransactionRowShowsPossibleDuplicateWhenFlagged(t *testing.T) {
	data := baseRowData()
	data["Source"] = "manual"
	data["IsDuplicate"] = true
	out := renderTransactionRow(t, data)
	if !strings.Contains(out, "possible duplicate") {
		t.Errorf("renderTransactionRow(IsDuplicate=true) = %q, want it to contain %q", out, "possible duplicate")
	}
}

// TestTransactionRowOmitsPossibleDuplicateWhenDataHasNoSuchKey covers the
// single-row fragments (create/edit/cancel-edit) on purpose: they build their
// map without an "IsDuplicate" entry at all, because a lone row has no
// siblings to compare against. A template map lookup that misses reads as
// false rather than erroring, which is what this asserts stays true.
func TestTransactionRowOmitsPossibleDuplicateWhenDataHasNoSuchKey(t *testing.T) {
	data := baseRowData()
	data["Source"] = "manual"
	out := renderTransactionRow(t, data)
	if strings.Contains(out, "possible duplicate") {
		t.Errorf("renderTransactionRow(no IsDuplicate key) = %q, want no %q mark", out, "possible duplicate")
	}
}
