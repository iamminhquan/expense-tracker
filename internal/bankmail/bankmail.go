// Package bankmail turns a forwarded bank notification email into a Notice
// -- a struct describing one transaction. It touches neither the database
// nor the network, which is what lets every rule about reading an email be
// tested without Postgres, the same split internal/csvimport draws for CSV
// files.
package bankmail

import (
	"errors"
	"strings"
	"time"
)

// ErrUnknownSender means from does not belong to a bank this app reads.
// Checked before the body is parsed at all, since a forged sender must never
// reach the parsing rules meant only for a real bank's mail.
var ErrUnknownSender = errors.New("sender is not a bank this app reads")

// ErrNotANotice means the sender is a known bank but the body does not match
// any transaction-notice shape this package recognizes -- an OTP message, an
// ad, a notice whose transaction failed, or one missing a field Parse needs
// to place the transaction. Returned rather than guessed at, because a
// guessed transaction is worse than none: it puts invented money into a real
// ledger.
var ErrNotANotice = errors.New("not a transaction notice")

// Notice is one bank transaction extracted from an email. Amount is the
// integer VND value -- this app stores and displays money as whole đồng,
// never decimal xu. Direction is "expense" or "income".
type Notice struct {
	Bank        string
	Amount      int64
	Direction   string
	OccurredAt  time.Time
	Description string
	// DebitAccount and BeneficiaryAccount are the two account numbers the
	// notice names, digits only, empty when the template did not carry one.
	//
	// They exist for one caller: deciding whether a transfer merely moved
	// money between two accounts the same person owns, which is neither
	// income nor expense in a single-ledger app. Reading them here rather
	// than in the handler keeps every rule about what an email says inside
	// this package -- the handler decides what to do about it, not what it
	// means.
	DebitAccount       string
	BeneficiaryAccount string
}

// bankSenders maps the domain suffix a bank's mail arrives from to the
// parser that reads its notices. Matching is on suffix rather than the whole
// address for the same reason internal/handlers/inbox_webhook.go's
// isKnownBankSender does: banks send from several local parts and change
// them without notice.
var bankSenders = map[string]func(subject, body string) (Notice, error){
	"@mbbank.com.vn": parseMB,
}

// vietnamLocation is Asia/Ho_Chi_Minh, loaded once and reused by every
// parser in this package -- every timestamp a bank notice carries is local
// Vietnam time. Mirrors internal/handlers/req_month.go's vietnamLocation
// rather than importing the handlers package, since internal/bankmail must
// stay a pure package that touches neither a request nor the database. If
// the timezone database isn't available in the runtime environment, fall
// back to a fixed UTC+7 offset (Vietnam has no DST) rather than failing.
var vietnamLocation = loadVietnamLocation()

func loadVietnamLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}

// Parse turns one forwarded email into a Notice. from is checked against a
// known bank's sending domain before subject or body is read at all: an
// unrecognized sender must return ErrUnknownSender regardless of what its
// body contains, since only the sender check stands between this function
// and an attacker-forged transaction email.
func Parse(from, subject, body string) (Notice, error) {
	addr := strings.ToLower(strings.TrimSpace(from))
	if i := strings.LastIndex(addr, "<"); i >= 0 {
		addr = strings.TrimSuffix(addr[i+1:], ">")
	}
	for suffix, parse := range bankSenders {
		if strings.HasSuffix(addr, suffix) {
			return parse(subject, body)
		}
	}
	return Notice{}, ErrUnknownSender
}
