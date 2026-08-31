package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"expensetracker/internal/handlers"
	"expensetracker/internal/inboxproc"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"
)

// mbCorrectionNotice builds an MB transfer notice whose only varying part
// is the reference code inside the transfer text -- which is what a real
// notice varies too, and which bankmail.NoteKey drops as a digit-bearing
// token. Two notices built here therefore describe two different transfers
// that fold to one hint key, which is the whole thing this test needs.
//
// A compact stand-in for the real sample internal/bankmail and
// internal/inboxproc test against, since what matters here is only that
// both bodies parse.
func mbCorrectionNotice(reference string) string {
	return `Cảm ơn Quý khách đã sử dụng dịch vụ MB eBanking.

 Ngày,
 giờ giao dịch

 31-08-2026 09:59:41

 Tài
 khoản trích nợ

 BUI MINH QUAN - 0011223344 (VND)

 Người
 thụ hưởng

 Nguyen Van A - 9988776655

 Số
 tiền giao dịch

 (VND) 50,000.00

 Nội
 dung chuyển tiền

 NGUYEN VAN A chuyen tien ` + reference + `

 Tình
 trạng

 Giao dịch thành công

Xin chân thành cảm ơn.
`
}

// pendingBankEmail stores one message the way inboxWebhookHandler does,
// without going through the Worker's signed POST: this test is about what
// happens after a message is stored, not about how it arrived.
func pendingBankEmail(t *testing.T, deps handlers.Deps, userID int64, messageID, body string) sqlcgen.BankEmail {
	t.Helper()
	email, err := deps.Queries.CreateBankEmail(context.Background(), sqlcgen.CreateBankEmailParams{
		UserID:      userID,
		MessageID:   messageID,
		FromAddress: "mbebanking@mbbank.com.vn",
		Subject:     "Thong bao giao dich",
		Body:        body,
		Status:      "pending",
	})
	if err != nil {
		t.Fatalf("CreateBankEmail(%q) error = %v", messageID, err)
	}
	return email
}

// TestCorrectingAnAutoFiledCategoryTeachesTheNextEmail is the end-to-end
// proof that the two halves of the learning mechanism agree on one key: the
// processor writes a row's description, and updateTransactionHandler derives
// the hint's note key from that stored description. Each half has its own
// unit test; only this one fails if they ever key on different strings --
// which is exactly what a change to the note truncation on either side
// would cause, silently, with every individual test still green.
//
// It lives in package handlers_test rather than beside the rest of the
// processing suite because the correction is a real PATCH through the same
// route the edit form posts to, and internal/inboxproc's own tests cannot
// import the router without an import cycle.
func TestCorrectingAnAutoFiledCategoryTeachesTheNextEmail(t *testing.T) {
	deps := newTestDeps(t)
	router := handlers.NewRouter(deps)
	cookie := loginAndGetCookie(t, router, deps, "email-correction@example.com", "s3cret-pass")

	user, err := deps.Queries.GetUserByEmail(context.Background(), "email-correction@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}
	corrected := personalCategory(t, deps, user.ID, "Corrected Transfer")
	processor := inboxproc.New(deps.Queries, deps.Classifier)

	firstEmail := pendingBankEmail(t, deps, user.ID, "<"+t.Name()+"-1@mail>", mbCorrectionNotice("26083101055223730"))
	processor.ProcessPending(user.ID)

	// Nothing has been corrected yet, so the first row must land on the
	// shared Other default -- if it landed anywhere else, the correction
	// below would prove nothing.
	otherCategory, err := deps.Queries.GetCategoryBySlug(context.Background(), pgval.Text("other"))
	if err != nil {
		t.Fatalf("GetCategoryBySlug(\"other\") error = %v", err)
	}
	var firstTxnID, firstCategoryID int64
	var firstDescription string
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT id, description, category_id FROM transactions WHERE bank_email_id = $1", firstEmail.ID,
	).Scan(&firstTxnID, &firstDescription, &firstCategoryID); err != nil {
		t.Fatalf("query first transaction: %v", err)
	}
	if firstCategoryID != otherCategory.ID {
		t.Fatalf("first transaction category_id = %d, want the Other fallback %d before any correction exists", firstCategoryID, otherCategory.ID)
	}

	tok := csrfTokenFor(t, router)
	form := url.Values{
		"category_id": {strconv.FormatInt(corrected.ID, 10)},
		"amount":      {"20000"},
		"description": {firstDescription},
		"occurred_on": {"2026-08-31"},
	}
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/transactions/%d", firstTxnID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	withCSRF(req, tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /transactions/%d = %d, want 200: %s", firstTxnID, rec.Code, rec.Body.String())
	}

	// A different reference number than the first email's, same transfer
	// text -- bankmail.NoteKey must fold both to the same key for this to
	// mean anything.
	secondEmail := pendingBankEmail(t, deps, user.ID, "<"+t.Name()+"-2@mail>", mbCorrectionNotice("26083101099999999"))
	processor.ProcessPending(user.ID)

	var secondCategoryID int64
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT category_id FROM transactions WHERE bank_email_id = $1", secondEmail.ID,
	).Scan(&secondCategoryID); err != nil {
		t.Fatalf("query second transaction's category: %v", err)
	}
	if secondCategoryID != corrected.ID {
		t.Errorf("second email's transaction category_id = %d, want %d (the corrected category, learned from the first)", secondCategoryID, corrected.ID)
	}
}
