package bankmail_test

import (
	"errors"
	"testing"

	"expensetracker/internal/bankmail"
)

// TestParseEmptySenderIsUnknown guards the check-sender-before-body order:
// an empty From should never fall through to body parsing.
func TestParseEmptySenderIsUnknown(t *testing.T) {
	_, err := bankmail.Parse("", "subject", "body")
	if !errors.Is(err, bankmail.ErrUnknownSender) {
		t.Errorf("Parse(empty sender) error = %v, want errors.Is ErrUnknownSender", err)
	}
}

// TestSentinelErrorStrings enforces the repo convention: error strings are
// lowercase and unpunctuated.
func TestSentinelErrorStrings(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrUnknownSender", bankmail.ErrUnknownSender, "sender is not a bank this app reads"},
		{"ErrNotANotice", bankmail.ErrNotANotice, "not a transaction notice"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("%s.Error() = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
