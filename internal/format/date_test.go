package format_test

import (
	"testing"
	"time"

	"expensetracker/internal/format"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestDateFormatting(t *testing.T) {
	d := pgtype.Date{Time: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC), Valid: true}
	if got := format.DateShort(d); got != "11 Aug" {
		t.Errorf("DateShort = %q, want 11 Aug", got)
	}
	if got := format.DateShort(pgtype.Date{}); got != "" {
		t.Errorf("DateShort(invalid) = %q, want empty string", got)
	}
	if got := format.DateLong(d); got != "11 Aug 2026" {
		t.Errorf("DateLong = %q, want 11 Aug 2026", got)
	}
	if got := format.DateLong(pgtype.Date{}); got != "" {
		t.Errorf("DateLong(invalid) = %q, want empty string", got)
	}
}
