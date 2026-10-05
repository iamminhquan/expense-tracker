package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/txnrule"
)

// txnForm is the add/edit transaction as the server reads it off a request:
// the four fields the quick-add form, the mobile sheet and the inline edit
// row all submit under the same names.
//
// It parses strictly, unlike the rest of the req_ group. Those value objects
// describe a view -- a malformed month or page is a view the user did not
// mean, and the lenient answer is the one they did. These four are what a
// row gets written from, so a value that will not read is a 400 rather than
// a default that quietly banks the wrong number.
type txnForm struct {
	CategoryID  int64
	Amount      int64
	OccurredOn  time.Time
	Description string
}

// txnFormFromRequest parses the four shared fields, returning the message
// for the first one that will not read and "" when all of them did.
func txnFormFromRequest(r *http.Request) (txnForm, string) {
	categoryID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		return txnForm{}, "invalid category"
	}
	amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if err != nil || amount <= 0 {
		return txnForm{}, "invalid amount"
	}
	occurredOn, err := time.Parse("2006-01-02", r.FormValue("occurred_on"))
	if err != nil {
		return txnForm{}, "invalid date"
	}
	return txnForm{
		CategoryID:  categoryID,
		Amount:      amount,
		OccurredOn:  occurredOn,
		Description: r.FormValue("description"),
	}, ""
}

// violation reports what to tell the user about a form that parsed but
// cannot be saved as a transaction of type txnType in a category of type
// categoryType, or "" when it can be.
//
// These three are separated from the parse failures above because they are
// answered differently: a value that will not parse never came from the
// form the app rendered, so it gets a status code, while these are things a
// person can do by hand and are shown back to them next to the field. The
// two limits come from internal/txnrule, which the CSV importer enforces
// too.
func (f txnForm) violation(categoryType, txnType string) string {
	switch {
	case categoryType != txnType:
		return "That category does not match the transaction type"
	case txnrule.NoteTooLong(f.Description):
		return fmt.Sprintf("Note must be %d characters or fewer", txnrule.MaxNoteRunes)
	case txnrule.TooFarInFuture(f.OccurredOn, time.Now().In(vietnamLocation)):
		return "That date is too far in the future"
	}
	return ""
}
