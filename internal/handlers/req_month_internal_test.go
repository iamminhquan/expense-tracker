package handlers

import (
	"testing"
	"time"
)

// TestVietnamLocationOffset covers Finding 6 from the final whole-branch
// review: month-boundary math was anchored to time.UTC instead of Vietnam's
// timezone (UTC+7). This asserts vietnamLocation actually carries a +7h
// offset, whether it resolved via the system's tzdata ("Asia/Ho_Chi_Minh")
// or the FixedZone fallback.
func TestVietnamLocationOffset(t *testing.T) {
	_, offset := time.Now().In(vietnamLocation).Zone()
	if offset != 7*60*60 {
		t.Fatalf("expected Vietnam location offset of +7h (25200s), got %ds", offset)
	}
}

// TestCurrentMonthRangeIsOneMonthWide asserts currentMonthRange returns a
// [from, to) pair starting on the 1st of the (Vietnam-local) current month
// and spanning exactly one calendar month.
func TestCurrentMonthRangeIsOneMonthWide(t *testing.T) {
	from, to := currentMonthRange()
	if !from.Valid || !to.Valid {
		t.Fatal("expected valid from/to dates")
	}
	if from.Time.Day() != 1 {
		t.Fatalf("expected from to be the 1st of the month, got day %d", from.Time.Day())
	}
	expectedTo := from.Time.AddDate(0, 1, 0)
	if !to.Time.Equal(expectedTo) {
		t.Fatalf("expected to = from + 1 calendar month (%v), got %v", expectedTo, to.Time)
	}
}

// TestCurrentMonthRangeMatchesVietnamNow asserts the range reflects "this
// month" as measured in Vietnam's timezone, not the server's local/UTC
// clock -- the specific defect Finding 6 flagged (a transaction added by a
// Vietnamese user near a month boundary could fall outside a UTC-anchored
// range).
func TestCurrentMonthRangeMatchesVietnamNow(t *testing.T) {
	now := time.Now().In(vietnamLocation)
	from, _ := currentMonthRange()
	if from.Time.Year() != now.Year() || from.Time.Month() != now.Month() {
		t.Fatalf("expected current month range to match Vietnam's current year/month (%v), got from=%v", now, from.Time)
	}
}
