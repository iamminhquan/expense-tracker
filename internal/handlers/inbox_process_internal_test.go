package handlers

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"
	"testing"

	"expensetracker/internal/auth"
	"expensetracker/internal/database"
	"expensetracker/internal/mailer"
	"expensetracker/internal/sqlcgen"
	"expensetracker/internal/web"

	"github.com/jackc/pgx/v5/pgtype"
)

// processTestDeps builds the same Deps the handler tests do, but lives here
// (package handlers, not handlers_test) because processPendingEmails is
// unexported and this suite has to call it directly rather than through a
// route. Duplicated rather than shared with auth_handlers_test.go's
// newTestDeps: that helper lives in package handlers_test, which a file in
// package handlers cannot import.
func processTestDeps(t *testing.T) Deps {
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

	templates, err := web.Templates(TemplateFuncs())
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	return Deps{
		DB:                 pool,
		Queries:            sqlcgen.New(pool),
		Sessions:           auth.NewManager(sqlcgen.New(pool)),
		PasswordResets:     auth.NewPasswordResetManager(sqlcgen.New(pool)),
		EmailVerifications: auth.NewEmailVerificationManager(sqlcgen.New(pool)),
		Mailer:             mailer.New(mailer.Config{}),
		Templates:          templates,
		CookieName:         "session_id",
	}
}

// createProcessTestUser inserts a bare account directly through
// Queries.CreateUser rather than the /register handler: this suite never
// drives a request, it calls processPendingEmails straight, so all it needs
// is a row to hang bank_emails and transactions off of. Named deterministically
// after the running test so parallel tests never collide on the unique
// email/username, mirroring registerTestUser in auth_handlers_test.go.
func createProcessTestUser(t *testing.T, deps Deps) int64 {
	t.Helper()
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
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
func createPendingBankEmail(t *testing.T, deps Deps, userID int64, from, subject, body string) sqlcgen.BankEmail {
	t.Helper()
	email, err := deps.Queries.CreateBankEmail(context.Background(), sqlcgen.CreateBankEmailParams{
		UserID:      userID,
		MessageID:   "<" + t.Name() + "@mail>",
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
func countTransactionsForUser(t *testing.T, deps Deps, userID int64) int {
	t.Helper()
	var n int
	err := deps.DB.QueryRow(context.Background(),
		"SELECT count(*) FROM transactions WHERE user_id = $1", userID).Scan(&n)
	if err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	return n
}

func bankEmailStatus(t *testing.T, deps Deps, id int64) (status, reason string) {
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

	processPendingEmails(deps, userID)

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

	category, err := deps.Queries.GetCategoryBySlug(context.Background(), pgText(otherExpenseSlug))
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

	processPendingEmails(deps, userID)

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

	processPendingEmails(deps, userID)

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
	processOneBankEmail(ctx, deps, email.ID)
	processOneBankEmail(ctx, deps, email.ID)

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

	processPendingEmails(deps, userID)

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
