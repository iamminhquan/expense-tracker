package handlers

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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
		{-85000, "85,000₫"}, // vnd() shows magnitude only, never a sign
	}
	for _, tc := range cases {
		if got := vnd(tc.in); got != tc.want {
			t.Errorf("vnd(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVNDSigned(t *testing.T) {
	if got := vndSigned(85000, "expense"); got != "-85,000₫" {
		t.Errorf("vndSigned(85000, expense) = %q, want -85,000₫", got)
	}
	if got := vndSigned(18500000, "income"); got != "+18,500,000₫" {
		t.Errorf("vndSigned(18500000, income) = %q, want +18,500,000₫", got)
	}
}

// The balance is a running wallet figure, not a delta, so a positive one
// carries no sign -- "+12,450,000₫" would read as an increase of that much.
// A negative balance still needs its minus, which is why vnd alone won't do.
func TestVNDBalance(t *testing.T) {
	if got := vndBalance(120000); got != "120,000₫" {
		t.Errorf("vndBalance(120000) = %q, want 120,000₫", got)
	}
	if got := vndBalance(-45000); got != "-45,000₫" {
		t.Errorf("vndBalance(-45000) = %q, want -45,000₫", got)
	}
	if got := vndBalance(0); got != "0₫" {
		t.Errorf("vndBalance(0) = %q, want 0₫", got)
	}
}

func TestDateFormatting(t *testing.T) {
	d := pgtype.Date{Time: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), Valid: true}
	if got := dateFull(d); got != "11 Aug 2026" {
		t.Errorf("dateFull = %q, want 11 Aug 2026", got)
	}
	if got := dateShort(d); got != "11 Aug" {
		t.Errorf("dateShort = %q, want 11 Aug", got)
	}
	if got := dateFull(pgtype.Date{}); got != "" {
		t.Errorf("dateFull(invalid) = %q, want empty string", got)
	}
}

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
		if got := countOf(tc.in, "transaction"); got != tc.want {
			t.Errorf("countOf(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
