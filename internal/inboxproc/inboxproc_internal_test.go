package inboxproc

import (
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"expensetracker/internal/bankmail"
	"expensetracker/internal/classify"
	"expensetracker/internal/database"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// procTest is the slice of a running deployment this suite needs: a pool to
// set fixtures up and read results back through, the queries a Processor
// runs on, and the classifier a test can swap for a fake server before the
// run it is about to trigger. It stands in for the whole handlers.Deps this
// suite built while it lived in that package, of which it only ever used
// these three fields.
type procTest struct {
	DB         *pgxpool.Pool
	Queries    *sqlcgen.Queries
	Classifier *classify.Classifier
}

// processor builds the Processor as procTest stands right now, rather than
// once at setup: a test that points Classifier at a fake server does so
// after processTestDeps has returned, and the very next run has to see it.
func (p procTest) processor() *Processor { return New(p.Queries, p.Classifier) }

// processTestDeps opens the pool every test here shares.
func processTestDeps(t *testing.T) procTest {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := database.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return procTest{
		DB:         pool,
		Queries:    sqlcgen.New(pool),
		Classifier: classify.New(classify.Config{}),
	}
}

// createProcessTestUser inserts a bare account directly through
// Queries.CreateUser rather than the /register handler: this suite never
// drives a request, it calls the Processor straight, so all it needs
// is a row to hang bank_emails and transactions off of. Named deterministically
// after the running test (hashing t.Name()) so parallel tests never collide
// on the unique email/username, mirroring registerTestUser in
// auth_handlers_test.go.
//
// suffix is optional and exists only for a test that needs two distinct
// accounts of its own, such as classify falling back on an id belonging to
// someone else's category: t.Name() alone is the same string on every call
// within one test, and each call opens by deleting any row already at the
// email it computes, so a second plain call would delete the first user's
// row out from under it (a real bug this suite hit once). Passing a second
// call site's own suffix -- e.g. "-other" -- folds a second value into the
// hash so the two calls land on different emails and neither is at risk of
// deleting the other's row.
func createProcessTestUser(t *testing.T, deps procTest, suffix ...string) int64 {
	t.Helper()
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	for _, s := range suffix {
		h.Write([]byte(s))
	}
	username := "inboxproc" + strconv.FormatUint(uint64(h.Sum32()), 36)
	email := username + "@example.com"
	deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })

	user, err := deps.Queries.CreateUser(context.Background(), sqlcgen.CreateUserParams{
		Email:        email,
		PasswordHash: "not-a-real-hash",
		Name:         "Inbox Process Test",
		Username:     username,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	return user.ID
}

// mbNotice is the same real MB eBanking transfer notice bankmail's own tests
// parse (internal/bankmail/mb_test.go's mbTransferNotice), reused here so
// the processing loop is exercised against the one real sample this app
// has. Amount: 20,000 VND, expense, occurred 31-08-2026 01:05:33 +07.
const mbNotice = `Cảm ơn Quý khách đã sử dụng dịch vụ MB eBanking.

MB xin thông báo giao dịch của Quý khách đã được thực hiện như sau:



 Ngày,
 giờ giao dịch



 31-08-2026 01:05:33






 Loại
 giao dịch



 Chuyển tiền nội bộ MB






 Số
 tham chiếu



 26083101055223730






 Tài
 khoản trích nợ



 NGUYEN VAN A - 0001111111111 (VND)






 Người
 thụ hưởng



 Nguyen Van B - 0399999999






 Số
 tiền giao dịch



 (VND) 20,000.00






 Nội
 dung chuyển tiền



 NGUYEN VAN A chuyen tien






 Cách
 thức lệnh



 Thanh toán ngay






 Tình
 trạng



 Giao dịch thành công
`

// createPendingBankEmail inserts one 'pending' bank_emails row with a
// caller-chosen sender/body, standing in for a forwarded message the
// webhook already stored. Returns the row so a test can act on its id.
// suffix distinguishes several emails created inside one test: the
// Message-ID is otherwise derived from the test name alone, and
// CreateBankEmail's ON CONFLICT DO NOTHING would swallow the second one.
func createPendingBankEmail(t *testing.T, deps procTest, userID int64, from, subject, body string, suffix ...string) sqlcgen.BankEmail {
	t.Helper()
	email, err := deps.Queries.CreateBankEmail(context.Background(), sqlcgen.CreateBankEmailParams{
		UserID:      userID,
		MessageID:   "<" + t.Name() + strings.Join(suffix, "") + "@mail>",
		FromAddress: from,
		Subject:     subject,
		Body:        body,
		Status:      "pending",
	})
	if err != nil {
		t.Fatalf("CreateBankEmail() error = %v", err)
	}
	return email
}

// countTransactionsForUser is a small direct query rather than a sqlc one:
// it exists purely so a test can assert "exactly one row", which no
// production handler needs.
func countTransactionsForUser(t *testing.T, deps procTest, userID int64) int {
	t.Helper()
	var n int
	err := deps.DB.QueryRow(context.Background(),
		"SELECT count(*) FROM transactions WHERE user_id = $1", userID).Scan(&n)
	if err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	return n
}

func bankEmailStatus(t *testing.T, deps procTest, id int64) (status, reason string) {
	t.Helper()
	err := deps.DB.QueryRow(context.Background(),
		"SELECT status, failure_reason FROM bank_emails WHERE id = $1", id).Scan(&status, &reason)
	if err != nil {
		t.Fatalf("query bank email %d status: %v", id, err)
	}
	return status, reason
}

// TestProcessPendingEmailsCreatesATransactionFromAValidMBNotice is the
// straight-line path: a stored MB transfer notice becomes exactly one
// transaction, flagged as coming from email and filed under the 'other'
// default since no category-hint memory exists yet.
func TestProcessPendingEmailsCreatesATransactionFromAValidMBNotice(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)

	deps.processor().ProcessPending(userID)

	if n := countTransactionsForUser(t, deps, userID); n != 1 {
		t.Fatalf("transactions after processing = %d, want 1", n)
	}

	var amount int64
	var source string
	var categoryID int64
	var bankEmailID pgtype.Int8
	err := deps.DB.QueryRow(context.Background(),
		"SELECT amount, source, category_id, bank_email_id FROM transactions WHERE user_id = $1", userID,
	).Scan(&amount, &source, &categoryID, &bankEmailID)
	if err != nil {
		t.Fatalf("query created transaction: %v", err)
	}
	if amount != 20000 {
		t.Errorf("transaction amount = %d, want %d", amount, 20000)
	}
	if source != "email" {
		t.Errorf("transaction source = %q, want %q", source, "email")
	}
	if !bankEmailID.Valid || bankEmailID.Int64 != email.ID {
		t.Errorf("transaction bank_email_id = %v, want %d", bankEmailID, email.ID)
	}

	category, err := deps.Queries.GetCategoryBySlug(context.Background(), pgval.Text(otherExpenseSlug))
	if err != nil {
		t.Fatalf("GetCategoryBySlug(%q) error = %v", otherExpenseSlug, err)
	}
	if categoryID != category.ID {
		t.Errorf("transaction category_id = %d, want %d (slug %q)", categoryID, category.ID, otherExpenseSlug)
	}

	status, _ := bankEmailStatus(t, deps, email.ID)
	if status != "imported" {
		t.Errorf("bank email status = %q, want %q", status, "imported")
	}

	var processedAt pgtype.Timestamptz
	err = deps.DB.QueryRow(context.Background(),
		"SELECT processed_at FROM bank_emails WHERE id = $1", email.ID).Scan(&processedAt)
	if err != nil {
		t.Fatalf("query processed_at: %v", err)
	}
	if !processedAt.Valid {
		t.Errorf("bank email processed_at = %v, want a set timestamp", processedAt)
	}
}

// TestProcessPendingEmailsIgnoresAnUnknownSender covers a sender the webhook
// itself accepts (it matches one of inbox_webhook.go's guessed TPBank
// domains, bankSenders) but that bankmail.Parse does not recognize -- no
// TPBank parser exists yet. The email must end up 'ignored', and nothing
// must be created.
func TestProcessPendingEmailsIgnoresAnUnknownSender(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	email := createPendingBankEmail(t, deps, userID, "notify@tpb.com.vn", "Bien dong so du", "some tpbank body")

	deps.processor().ProcessPending(userID)

	if n := countTransactionsForUser(t, deps, userID); n != 0 {
		t.Errorf("transactions after processing unknown sender = %d, want 0", n)
	}
	status, reason := bankEmailStatus(t, deps, email.ID)
	if status != "ignored" {
		t.Errorf("bank email status = %q, want %q", status, "ignored")
	}
	if reason == "" {
		t.Error("bank email failure_reason after ignoring unknown sender is empty, want a reason")
	}
}

// TestProcessPendingEmailsIgnoresANonNoticeBody covers a known bank sending
// something that is not a transaction notice (an ad, an OTP): the shape
// bankmail.ErrNotANotice exists for. Must also land on 'ignored', not
// 'failed' -- this is routine mail, not a bug.
func TestProcessPendingEmailsIgnoresANonNoticeBody(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	body := "Cam on quy khach da su dung dich vu MB eBanking. Uu dai thang nay danh cho ban!"
	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Uu dai thang 8", body)

	deps.processor().ProcessPending(userID)

	if n := countTransactionsForUser(t, deps, userID); n != 0 {
		t.Errorf("transactions after processing non-notice body = %d, want 0", n)
	}
	status, _ := bankEmailStatus(t, deps, email.ID)
	if status != "ignored" {
		t.Errorf("bank email status = %q, want %q", status, "ignored")
	}
}

// TestProcessPendingEmailsTwiceCreatesOnlyOneTransaction is the claim test:
// processing the same email a second time must not create a second
// transaction. It calls processOneBankEmail directly at the same id rather
// than routing a second pass through processPendingEmails, because
// ListPendingBankEmailIDs already filters to 'pending' rows on its own -- a
// second call to processPendingEmails would have its list step quietly skip
// the now-'imported' email, which would pass this test even if
// ClaimPendingBankEmail's own guard were broken. Calling processOneBankEmail
// straight is what actually exercises the claim: after the first call
// leaves the row 'imported', a second call whose claim ignored the
// UPDATE ... WHERE status = 'pending' RETURNING guard would still match on
// id alone and create a second transaction.
func TestProcessPendingEmailsTwiceCreatesOnlyOneTransaction(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)

	ctx := context.Background()
	deps.processor().processOne(ctx, email.ID)
	deps.processor().processOne(ctx, email.ID)

	if n := countTransactionsForUser(t, deps, userID); n != 1 {
		t.Errorf("transactions after processing the same email twice = %d, want 1", n)
	}
	status, _ := bankEmailStatus(t, deps, email.ID)
	if status != "imported" {
		t.Errorf("bank email status after processing twice = %q, want %q", status, "imported")
	}
}

// TestProcessPendingEmailsMarksAnOtherFailureAsFailed exercises the second
// row of the error-mapping table: an error that is neither ErrUnknownSender
// nor ErrNotANotice -- our own bug, not routine mail -- must land the email
// on 'failed' with the real error text recorded, and create nothing.
// bankmail.Parse succeeds here (a known sender, a real notice shape); the
// failure is forced downstream by renaming the 'other' category's slug out
// from under GetCategoryBySlug for the duration of the test, so
// createTransactionFromNotice fails the same way a misconfigured deployment
// would -- exactly the kind of failure processOneBankEmail's unconditional
// MarkBankEmailFailed branch (as opposed to markUnprocessableBankEmail's
// sender/shape branch) exists to catch.
func TestProcessPendingEmailsMarksAnOtherFailureAsFailed(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)

	// The 'other' category row is shared by every other test (and by the
	// running app schema), so this only ever renames its slug and always
	// restores it via t.Cleanup -- never deletes the row.
	ctx := context.Background()
	const disabledSlug = "other-disabled-for-test"
	if _, err := deps.DB.Exec(ctx,
		"UPDATE categories SET slug = $1 WHERE slug = $2", disabledSlug, otherExpenseSlug); err != nil {
		t.Fatalf("rename %q category slug: %v", otherExpenseSlug, err)
	}
	t.Cleanup(func() {
		if _, err := deps.DB.Exec(context.Background(),
			"UPDATE categories SET slug = $1 WHERE slug = $2", otherExpenseSlug, disabledSlug); err != nil {
			t.Errorf("restore %q category slug: %v", otherExpenseSlug, err)
		}
	})

	deps.processor().ProcessPending(userID)

	if n := countTransactionsForUser(t, deps, userID); n != 0 {
		t.Errorf("transactions after a non-sender/shape failure = %d, want 0", n)
	}
	status, reason := bankEmailStatus(t, deps, email.ID)
	if status != "failed" {
		t.Errorf("bank email status = %q, want %q", status, "failed")
	}
	if reason == "" {
		t.Error("bank email failure_reason after a non-sender/shape failure is empty, want the underlying error")
	}
}

// processTestCategory creates one category owned by userID, mirroring
// handlers_test's own personalCategory helper -- duplicated rather than
// shared because that one lives in another package's test binary.
func processTestCategory(t *testing.T, deps procTest, userID int64, name, typ string) sqlcgen.Category {
	t.Helper()
	category, err := deps.Queries.CreateCategory(context.Background(), sqlcgen.CreateCategoryParams{
		UserID: pgtype.Int8{Int64: userID, Valid: true}, Name: name, Type: typ, Color: "#D97757",
	})
	if err != nil {
		t.Fatalf("create category %q: %v", name, err)
	}
	return category
}

// TestProcessPendingEmailsUsesCategoryHintWhenNoteKeyMatches is the hit
// half of step 6: a category_hints row for this user and this notice's own
// note key must decide the category instead of the Other/Other income
// fallback, with no AI call involved (none is even configured in this test
// -- if the hint weren't consulted first, this would just fall back to
// Other and go undetected as it did before this slice).
func TestProcessPendingEmailsUsesCategoryHintWhenNoteKeyMatches(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)

	// bankmail.Parse the same notice the plain fallback test uses, so the
	// hint is planted under exactly the note key the processing loop will
	// itself compute -- guessing the key independently would let this test
	// pass or fail for the wrong reason if NoteKey's own rule ever changed.
	notice, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
	if err != nil {
		t.Fatalf("bankmail.Parse() error = %v", err)
	}
	remembered := processTestCategory(t, deps, userID, "Remembered Transfer", "expense")
	if _, err := deps.Queries.UpsertCategoryHint(context.Background(), sqlcgen.UpsertCategoryHintParams{
		UserID: userID, NoteKey: bankmail.NoteKey(notice.Description), CategoryID: remembered.ID,
	}); err != nil {
		t.Fatalf("UpsertCategoryHint() error = %v", err)
	}

	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
	deps.processor().ProcessPending(userID)

	if n := countTransactionsForUser(t, deps, userID); n != 1 {
		t.Fatalf("transactions after processing = %d, want 1", n)
	}
	var categoryID int64
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT category_id FROM transactions WHERE bank_email_id = $1", email.ID,
	).Scan(&categoryID); err != nil {
		t.Fatalf("query created transaction's category: %v", err)
	}
	if categoryID != remembered.ID {
		t.Errorf("transaction category_id = %d, want %d (the remembered hint, not the Other fallback)", categoryID, remembered.ID)
	}
}

// TestProcessPendingEmailsSurvivesACategoryHintLookupError pins the fix for
// review finding 1: a hint is an optimisation, never a requirement for the
// transaction to exist, so any error reading category_hints -- not just an
// ordinary miss -- must still create the transaction against the
// Other/Other income fallback rather than losing it. Forcing that error is
// done the same way TestProcessPendingEmailsMarksAnOtherFailureAsFailed
// forces its own downstream failure: rename the table out from under the
// query for the duration of the test and restore it via t.Cleanup, rather
// than reshaping resolveCategoryForNotice into something a mock could
// intercept.
func TestProcessPendingEmailsSurvivesACategoryHintLookupError(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)

	ctx := context.Background()
	if _, err := deps.DB.Exec(ctx, "ALTER TABLE category_hints RENAME TO category_hints_test_disabled"); err != nil {
		t.Fatalf("rename category_hints: %v", err)
	}
	t.Cleanup(func() {
		if _, err := deps.DB.Exec(context.Background(), "ALTER TABLE category_hints_test_disabled RENAME TO category_hints"); err != nil {
			t.Errorf("restore category_hints: %v", err)
		}
	})

	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
	deps.processor().ProcessPending(userID)

	if n := countTransactionsForUser(t, deps, userID); n != 1 {
		t.Fatalf("transactions after a category_hints lookup error = %d, want 1 (a classification failure must never cost a transaction)", n)
	}

	otherCategory, err := deps.Queries.GetCategoryBySlug(ctx, pgval.Text(otherExpenseSlug))
	if err != nil {
		t.Fatalf("GetCategoryBySlug(%q) error = %v", otherExpenseSlug, err)
	}
	var categoryID int64
	if err := deps.DB.QueryRow(ctx,
		"SELECT category_id FROM transactions WHERE bank_email_id = $1", email.ID,
	).Scan(&categoryID); err != nil {
		t.Fatalf("query created transaction's category: %v", err)
	}
	if categoryID != otherCategory.ID {
		t.Errorf("transaction category_id = %d, want the Other fallback %d", categoryID, otherCategory.ID)
	}

	status, _ := bankEmailStatus(t, deps, email.ID)
	if status != "imported" {
		t.Errorf("bank email status after a category_hints lookup error = %q, want %q (not 'failed')", status, "imported")
	}
}

// classifyAnswerJSON builds a fake generateContent response whose only text
// part is the structured-output JSON {"category_id":"id"} -- the shape
// classify.Classifier expects back, real or faked. The id travels as a
// string because the schema constrains it with a STRING enum, which is the
// only type Gemini documents enum for.
func classifyAnswerJSON(id int64) string {
	return fmt.Sprintf(`{
		"candidates": [{
			"content": {"role": "model", "parts": [{"text": %q}]},
			"finishReason": "STOP"
		}],
		"usageMetadata": {"promptTokenCount": 120, "candidatesTokenCount": 8},
		"modelVersion": "gemini-3.5-flash-lite"
	}`, fmt.Sprintf(`{"category_id":"%d"}`, id))
}

// fakeClassifyServer stands in for the Gemini API: every request
// increments the returned counter and gets handler's response. deps'
// Classifier must be pointed at it (via classify.Config.Endpoint) by the
// caller -- this alone does not wire anything into a Deps.
func fakeClassifyServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *int32) {
	t.Helper()
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

// fakeClassifyAnswering always answers with categoryID.
func fakeClassifyAnswering(categoryID int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, classifyAnswerJSON(categoryID))
	}
}

// fakeClassifyFailingWith always answers with the given non-2xx status.
func fakeClassifyFailingWith(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, `{"error":{"code":500,"message":"boom","status":"INTERNAL"}}`)
	}
}

// TestProcessPendingEmailsUsesTheClassifierWhenHintMisses is the slice 4
// straight-line path: no category_hints row exists yet, so the miss branch
// of resolveCategoryForNotice calls out to classify.Classifier, and the
// category it answers with -- not Other -- is what the transaction and the
// newly written hint both land on.
func TestProcessPendingEmailsUsesTheClassifierWhenHintMisses(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	picked := processTestCategory(t, deps, userID, "AI Picked Category", "expense")

	server, requests := fakeClassifyServer(t, fakeClassifyAnswering(picked.ID))
	deps.Classifier = classify.New(classify.Config{APIKey: "test-key", Endpoint: server.URL})

	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
	deps.processor().ProcessPending(userID)

	if got := atomic.LoadInt32(requests); got != 1 {
		t.Errorf("fake classify server received %d requests, want 1", got)
	}

	var categoryID int64
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT category_id FROM transactions WHERE bank_email_id = $1", email.ID,
	).Scan(&categoryID); err != nil {
		t.Fatalf("query created transaction's category: %v", err)
	}
	if categoryID != picked.ID {
		t.Errorf("transaction category_id = %d, want %d (the classifier's answer)", categoryID, picked.ID)
	}

	notice, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
	if err != nil {
		t.Fatalf("bankmail.Parse() error = %v", err)
	}
	hint, err := deps.Queries.GetCategoryHint(context.Background(), sqlcgen.GetCategoryHintParams{
		UserID: userID, NoteKey: bankmail.NoteKey(notice.Description),
	})
	if err != nil {
		t.Fatalf("GetCategoryHint() error = %v, want the hint the classification just wrote", err)
	}
	if hint.CategoryID != picked.ID {
		t.Errorf("written hint category_id = %d, want %d", hint.CategoryID, picked.ID)
	}
}

// TestProcessPendingEmailsHintFromClassifyStopsASecondCall is the test that
// proves the whole design pays for itself: a second email sharing the
// first's note key (a different reference number, same transfer text) must
// be decided from the hint the first call's classification wrote, not by
// asking Claude again.
func TestProcessPendingEmailsHintFromClassifyStopsASecondCall(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	picked := processTestCategory(t, deps, userID, "AI Picked Category", "expense")

	server, requests := fakeClassifyServer(t, fakeClassifyAnswering(picked.ID))
	deps.Classifier = classify.New(classify.Config{APIKey: "test-key", Endpoint: server.URL})

	createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
	deps.processor().ProcessPending(userID)
	if got := atomic.LoadInt32(requests); got != 1 {
		t.Fatalf("fake classify server received %d requests after the first email, want 1", got)
	}

	secondBody := strings.Replace(mbNotice, "26083101055223730", "26083101099999999", 1)
	secondEmail, err := deps.Queries.CreateBankEmail(context.Background(), sqlcgen.CreateBankEmailParams{
		UserID: userID, MessageID: "<" + t.Name() + "-2@mail>",
		FromAddress: "mbebanking@mbbank.com.vn", Subject: "Thong bao giao dich",
		Body: secondBody, Status: "pending",
	})
	if err != nil {
		t.Fatalf("CreateBankEmail() (second) error = %v", err)
	}
	deps.processor().ProcessPending(userID)

	if got := atomic.LoadInt32(requests); got != 1 {
		t.Errorf("fake classify server received %d requests after a second, same-note-key email, want 1 (the hint should have answered it)", got)
	}
	var secondCategoryID int64
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT category_id FROM transactions WHERE bank_email_id = $1", secondEmail.ID,
	).Scan(&secondCategoryID); err != nil {
		t.Fatalf("query second transaction's category: %v", err)
	}
	if secondCategoryID != picked.ID {
		t.Errorf("second email's transaction category_id = %d, want %d (from the hint, not a fresh classification)", secondCategoryID, picked.ID)
	}
}

// TestProcessPendingEmailsFallsBackWhenClassifierFails covers the rule that
// outranks the rest of slice 4: a classification failure must never cost a
// transaction. 429 and 500 are each exercised as subtests of the same
// shape -- both must still create the transaction, filed under the
// Other/Other income fallback, and processOneBankEmail must not panic.
func TestProcessPendingEmailsFallsBackWhenClassifierFails(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			deps := processTestDeps(t)
			userID := createProcessTestUser(t, deps)
			server, requests := fakeClassifyServer(t, fakeClassifyFailingWith(status))
			deps.Classifier = classify.New(classify.Config{APIKey: "test-key", Endpoint: server.URL})

			email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
			deps.processor().ProcessPending(userID)

			if got := atomic.LoadInt32(requests); got == 0 {
				t.Error("fake classify server received 0 requests, want at least 1 -- the classifier should have been called")
			}
			if n := countTransactionsForUser(t, deps, userID); n != 1 {
				t.Fatalf("transactions after a %d classify response = %d, want 1 (a classification failure must never cost a transaction)", status, n)
			}
			otherCategory, err := deps.Queries.GetCategoryBySlug(context.Background(), pgval.Text(otherExpenseSlug))
			if err != nil {
				t.Fatalf("GetCategoryBySlug(%q) error = %v", otherExpenseSlug, err)
			}
			var categoryID int64
			if err := deps.DB.QueryRow(context.Background(),
				"SELECT category_id FROM transactions WHERE bank_email_id = $1", email.ID,
			).Scan(&categoryID); err != nil {
				t.Fatalf("query created transaction's category: %v", err)
			}
			if categoryID != otherCategory.ID {
				t.Errorf("transaction category_id = %d, want the Other fallback %d", categoryID, otherCategory.ID)
			}
			status, _ := bankEmailStatus(t, deps, email.ID)
			if status != "imported" {
				t.Errorf("bank email status = %q, want %q (not 'failed')", status, "imported")
			}
		})
	}
}

// TestProcessPendingEmailsFallsBackWhenClassifierAnswerIsUntrustworthy
// covers the two shapes of a malformed answer that must never be trusted
// onto a transaction: an id that matches nothing at all, and an id that is
// real but belongs to a category this account was never offered (modeled
// here as a category owned outright by a different user). Both must fall
// back exactly like an API failure would.
func TestProcessPendingEmailsFallsBackWhenClassifierAnswerIsUntrustworthy(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	otherUserID := createProcessTestUser(t, deps, "-other")
	othersCategory := processTestCategory(t, deps, otherUserID, "Someone Else's Category", "expense")

	tests := map[string]int64{
		"no category has this id":           999_999_999,
		"id belongs to a different account": othersCategory.ID,
	}
	for name, answerID := range tests {
		t.Run(name, func(t *testing.T) {
			deps := deps
			server, requests := fakeClassifyServer(t, fakeClassifyAnswering(answerID))
			deps.Classifier = classify.New(classify.Config{APIKey: "test-key", Endpoint: server.URL})

			email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
			deps.processor().ProcessPending(userID)

			if got := atomic.LoadInt32(requests); got == 0 {
				t.Error("fake classify server received 0 requests, want at least 1")
			}
			otherCategory, err := deps.Queries.GetCategoryBySlug(context.Background(), pgval.Text(otherExpenseSlug))
			if err != nil {
				t.Fatalf("GetCategoryBySlug(%q) error = %v", otherExpenseSlug, err)
			}
			var categoryID int64
			if err := deps.DB.QueryRow(context.Background(),
				"SELECT category_id FROM transactions WHERE bank_email_id = $1", email.ID,
			).Scan(&categoryID); err != nil {
				t.Fatalf("query created transaction's category: %v", err)
			}
			if categoryID != otherCategory.ID {
				t.Errorf("transaction category_id = %d, want the Other fallback %d", categoryID, otherCategory.ID)
			}
		})
	}
}

// TestProcessPendingEmailsSkipsAnUnconfiguredClassifier is the last leg of
// the fallback contract: an unconfigured Classifier must not attempt a call
// at all (Configured() gates it before any network round trip), yet the
// transaction is still created under the Other fallback exactly as if
// classify.Classifier didn't exist.
func TestProcessPendingEmailsSkipsAnUnconfiguredClassifier(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	server, requests := fakeClassifyServer(t, fakeClassifyAnswering(1))
	// Deliberately no APIKey: Configured() must be false, and the fake
	// server above exists only to prove nothing ever reaches it.
	deps.Classifier = classify.New(classify.Config{Endpoint: server.URL})

	email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich", mbNotice)
	deps.processor().ProcessPending(userID)

	if got := atomic.LoadInt32(requests); got != 0 {
		t.Errorf("fake classify server received %d requests, want 0 -- an unconfigured Classifier must never call out", got)
	}
	if n := countTransactionsForUser(t, deps, userID); n != 1 {
		t.Fatalf("transactions after processing with an unconfigured classifier = %d, want 1", n)
	}
	otherCategory, err := deps.Queries.GetCategoryBySlug(context.Background(), pgval.Text(otherExpenseSlug))
	if err != nil {
		t.Fatalf("GetCategoryBySlug(%q) error = %v", otherExpenseSlug, err)
	}
	var categoryID int64
	if err := deps.DB.QueryRow(context.Background(),
		"SELECT category_id FROM transactions WHERE bank_email_id = $1", email.ID,
	).Scan(&categoryID); err != nil {
		t.Fatalf("query created transaction's category: %v", err)
	}
	if categoryID != otherCategory.ID {
		t.Errorf("transaction category_id = %d, want the Other fallback %d", categoryID, otherCategory.ID)
	}
}

// mbNoticeBetween builds an MB transfer notice moving money out of debit
// and into beneficiary, so a test can state the two account numbers that
// decide whether a notice is a self-transfer.
func mbNoticeBetween(debit, beneficiary string) string {
	return `Cảm ơn Quý khách đã sử dụng dịch vụ MB eBanking.

 Ngày,
 giờ giao dịch

 31-08-2026 09:59:41

 Tài
 khoản trích nợ

 BUI MINH QUAN - ` + debit + ` (VND)

 Người
 thụ hưởng

 Bui Minh Quan - ` + beneficiary + `

 Số
 tiền giao dịch

 (VND) 50,000.00

 Nội
 dung chuyển tiền

 BUI MINH QUAN chuyen tien

 Tình
 trạng

 Giao dịch thành công

Xin chân thành cảm ơn.
Website: http://www.mbbank.com.vn
`
}

// TestProcessIgnoresATransferBetweenTheOwnersOwnAccounts is the scenario
// that prompted this rule, replayed exactly: two MB accounts belonging to
// one person, and two notices that are mirror images of each other.
//
// The first notice teaches the app that the debited account is the owner's
// and is recorded as an expense -- nothing yet proves the payee is also
// theirs. The second, moving the money back, names that first account as
// the beneficiary, which *is* proof, and must be ignored rather than
// recorded: a pot that pays itself neither earned nor spent anything.
func TestProcessIgnoresATransferBetweenTheOwnersOwnAccounts(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	ctx := context.Background()

	const accountA, accountB = "0001111111111", "0399999999"

	first := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich",
		mbNoticeBetween(accountA, accountB), "first")
	deps.processor().processOne(ctx, first.ID)

	if n := countTransactionsForUser(t, deps, userID); n != 1 {
		t.Fatalf("transactions after the first notice = %d, want 1 -- nothing yet proves the payee is the owner's", n)
	}

	second := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn", "Thong bao giao dich",
		mbNoticeBetween(accountB, accountA), "second")
	deps.processor().processOne(ctx, second.ID)

	if n := countTransactionsForUser(t, deps, userID); n != 1 {
		t.Errorf("transactions after the mirrored notice = %d, want 1 -- money moved between two of the owner's own accounts", n)
	}
	status, reason := bankEmailStatus(t, deps, second.ID)
	if status != "ignored" {
		t.Errorf("mirrored notice status = %q, want %q", status, "ignored")
	}
	if reason != internalTransferReason {
		t.Errorf("mirrored notice reason = %q, want %q", reason, internalTransferReason)
	}
}

// TestProcessStillRecordsAPaymentToSomeoneElse is the guard on the rule
// above, and the more important of the two: ignoring a real payment removes
// money from the ledger that the owner has no way of noticing is gone.
// Paying the same payee twice must stay two expenses, however many notices
// have taught the app about the paying account.
func TestProcessStillRecordsAPaymentToSomeoneElse(t *testing.T) {
	deps := processTestDeps(t)
	userID := createProcessTestUser(t, deps)
	ctx := context.Background()

	const mine, payee = "0001111111111", "0999888777"

	for i := range 2 {
		email := createPendingBankEmail(t, deps, userID, "mbebanking@mbbank.com.vn",
			"Thong bao giao dich", mbNoticeBetween(mine, payee), strconv.Itoa(i))
		deps.processor().processOne(ctx, email.ID)
	}

	if n := countTransactionsForUser(t, deps, userID); n != 2 {
		t.Errorf("transactions after paying the same payee twice = %d, want 2 -- a payee is never proof of ownership", n)
	}
}
