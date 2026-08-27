package format_test

import (
	"testing"

	"expensetracker/internal/format"
)

func TestVND(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0₫"},
		{999, "999₫"},
		{50000, "50,000₫"},
		{18500000, "18,500,000₫"},
		{-85000, "85,000₫"}, // VND shows magnitude only, never a sign
	}
	for _, tc := range cases {
		if got := format.VND(tc.in); got != tc.want {
			t.Errorf("VND(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVNDSigned(t *testing.T) {
	if got := format.VNDSigned(85000, "expense"); got != "-85,000₫" {
		t.Errorf("VNDSigned(85000, expense) = %q, want -85,000₫", got)
	}
	if got := format.VNDSigned(18500000, "income"); got != "+18,500,000₫" {
		t.Errorf("VNDSigned(18500000, income) = %q, want +18,500,000₫", got)
	}
}

// The balance is a running wallet figure, not a delta, so a positive one
// carries no sign -- "+12,450,000₫" would read as an increase of that much.
// A negative balance still needs its minus, which is why VND alone won't do.
func TestVNDBalance(t *testing.T) {
	if got := format.VNDBalance(120000); got != "120,000₫" {
		t.Errorf("VNDBalance(120000) = %q, want 120,000₫", got)
	}
	if got := format.VNDBalance(-45000); got != "-45,000₫" {
		t.Errorf("VNDBalance(-45000) = %q, want -45,000₫", got)
	}
	if got := format.VNDBalance(0); got != "0₫" {
		t.Errorf("VNDBalance(0) = %q, want 0₫", got)
	}
}
