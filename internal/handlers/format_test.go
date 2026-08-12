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
		{50000, "50.000₫"},
		{18500000, "18.500.000₫"},
		{-85000, "85.000₫"}, // vnd() shows magnitude only, never a sign
	}
	for _, tc := range cases {
		if got := vnd(tc.in); got != tc.want {
			t.Errorf("vnd(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVNDSigned(t *testing.T) {
	if got := vndSigned(85000, "expense"); got != "-85.000₫" {
		t.Errorf("vndSigned(85000, expense) = %q, want -85.000₫", got)
	}
	if got := vndSigned(18500000, "income"); got != "+18.500.000₫" {
		t.Errorf("vndSigned(18500000, income) = %q, want +18.500.000₫", got)
	}
}

func TestVNDBalance(t *testing.T) {
	if got := vndBalance(120000); got != "+120.000₫" {
		t.Errorf("vndBalance(120000) = %q, want +120.000₫", got)
	}
	if got := vndBalance(-45000); got != "-45.000₫" {
		t.Errorf("vndBalance(-45000) = %q, want -45.000₫", got)
	}
}

func TestDateFormatting(t *testing.T) {
	d := pgtype.Date{Time: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), Valid: true}
	if got := dateFull(d); got != "11/08/2026" {
		t.Errorf("dateFull = %q, want 11/08/2026", got)
	}
	if got := dateShort(d); got != "11/08" {
		t.Errorf("dateShort = %q, want 11/08", got)
	}
	if got := dateFull(pgtype.Date{}); got != "" {
		t.Errorf("dateFull(invalid) = %q, want empty string", got)
	}
}
