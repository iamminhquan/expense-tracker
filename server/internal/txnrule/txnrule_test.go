package txnrule_test

import (
	"strings"
	"testing"
	"time"

	"expensetracker/internal/txnrule"
)

func TestNoteTooLong(t *testing.T) {
	tests := []struct {
		name string
		note string
		want bool
	}{
		{"empty", "", false},
		{"at the limit", strings.Repeat("a", txnrule.MaxNoteRunes), false},
		{"one over", strings.Repeat("a", txnrule.MaxNoteRunes+1), true},
		// Counted in runes: 200 Vietnamese characters are far more than 200
		// bytes, and rejecting them would be measuring the wrong thing.
		{"multi-byte at the limit", strings.Repeat("ế", txnrule.MaxNoteRunes), false},
		{"multi-byte one over", strings.Repeat("ế", txnrule.MaxNoteRunes+1), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := txnrule.NoteTooLong(tc.note); got != tc.want {
				t.Errorf("NoteTooLong(%d runes) = %v, want %v", len([]rune(tc.note)), got, tc.want)
			}
		})
	}
}

func TestTooFarInFuture(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		days int
		want bool
	}{
		{"long past", -400, false},
		{"today", 0, false},
		{"at the limit", txnrule.MaxFutureDays, false},
		{"one day past the limit", txnrule.MaxFutureDays + 1, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			occurredOn := now.AddDate(0, 0, tc.days)
			if got := txnrule.TooFarInFuture(occurredOn, now); got != tc.want {
				t.Errorf("TooFarInFuture(now%+dd, now) = %v, want %v", tc.days, got, tc.want)
			}
		})
	}
}
