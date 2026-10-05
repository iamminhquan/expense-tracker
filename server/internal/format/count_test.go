package format_test

import (
	"testing"

	"expensetracker/internal/format"
)

func TestCountOf(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{0, "0 transactions"},
		{1, "1 transaction"},
		{2, "2 transactions"},
		{int64(1), "1 transaction"},
		{int64(12), "12 transactions"},
	}
	for _, tc := range cases {
		if got := format.CountOf(tc.in, "transaction"); got != tc.want {
			t.Errorf("CountOf(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
