package handlers

import (
	"testing"
	"time"
)

func TestScopeReadsACalendarMonth(t *testing.T) {
	s := newTxnScope("2026-08")

	if s.All {
		t.Error("expected a named month not to be the all-time scope")
	}
	if s.Value != "2026-08" {
		t.Errorf("newTxnScope(%q).Value = %q, want %q", "2026-08", s.Value, "2026-08")
	}
	if s.Label != "August 2026" {
		t.Errorf("newTxnScope(%q).Label = %q, want %q", "2026-08", s.Label, "August 2026")
	}

	from, to := s.Bounds()
	if got := from.Time.Format("2006-01-02"); got != "2026-08-01" {
		t.Errorf("lower bound = %q, want %q", got, "2026-08-01")
	}
	if got := to.Time.Format("2006-01-02"); got != "2026-09-01" {
		t.Errorf("upper bound = %q, want %q", got, "2026-09-01")
	}
}

// The whole point of the all-time scope: bounds wide enough that the month
// predicate stops narrowing anything, so the one query serves both views.
func TestScopeSpansEveryMonthWhenAskedForAll(t *testing.T) {
	s := newTxnScope("all")

	if !s.All {
		t.Fatal("expected \"all\" to be the all-time scope")
	}
	if s.Value != "all" {
		t.Errorf("newTxnScope(%q).Value = %q, want %q", "all", s.Value, "all")
	}
	if s.Label != "All months" {
		t.Errorf("newTxnScope(%q).Label = %q, want %q", "all", s.Label, "All months")
	}

	from, to := s.Bounds()
	ancient := time.Date(1970, 3, 4, 0, 0, 0, 0, time.UTC)
	distant := time.Date(2999, 3, 4, 0, 0, 0, 0, time.UTC)
	if !from.Time.Before(ancient) {
		t.Errorf("lower bound %v does not reach back past %v", from.Time, ancient)
	}
	if !to.Time.After(distant) {
		t.Errorf("upper bound %v does not reach forward past %v", to.Time, distant)
	}
}

// Value is what every link the page builds carries in ?month=, so it has to
// survive the round trip -- a scope that rendered back as its lower bound
// would send "0001-01" into the pager and the export link.
func TestScopeValueSurvivesARoundTrip(t *testing.T) {
	for _, param := range []string{"2026-08", "all"} {
		if got := newTxnScope(newTxnScope(param).Value).Value; got != param {
			t.Errorf("round trip of %q gave %q", param, got)
		}
	}
}

// An unusable month falls back to the current one rather than erroring, the
// way monthRangeFor always has -- and it must not fall back to "all", which
// would quietly widen the list instead of narrowing it.
func TestScopeFallsBackToThisMonthNotToAll(t *testing.T) {
	thisMonth := time.Now().In(vietnamLocation).Format("2006-01")
	for _, param := range []string{"", "  ", "nonsense", "2026-13", "ALL"} {
		s := newTxnScope(param)
		if s.All {
			t.Errorf("newTxnScope(%q) widened the list to every month", param)
		}
		if s.Value != thisMonth {
			t.Errorf("newTxnScope(%q).Value = %q, want the current month %q", param, s.Value, thisMonth)
		}
	}
}

// The dashboard's charts are built month by month and have no meaning over a
// whole history, so monthRangeFor -- which every one of them still calls --
// must not learn about "all". It treats it as malformed, like any other
// unparseable month, and shows the current one.
func TestTheAllScopeStaysOutOfTheMonthRangeTheDashboardUses(t *testing.T) {
	from, to := monthRangeFor("all")
	currentFrom, currentTo := currentMonthRange()

	if !from.Time.Equal(currentFrom.Time) || !to.Time.Equal(currentTo.Time) {
		t.Errorf("monthRangeFor(\"all\") = [%v, %v), want the current month [%v, %v)",
			from.Time, to.Time, currentFrom.Time, currentTo.Time)
	}
}
