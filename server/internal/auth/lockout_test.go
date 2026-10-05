package auth_test

import (
	"testing"
	"time"

	"expensetracker/internal/auth"

	"github.com/jackc/pgx/v5/pgtype"
)

// pgTime builds the pgtype.Timestamptz that sqlc hands back for a nullable
// column, so the lockout tests can state "no lock recorded" as valid=false.
func pgTime(t time.Time, valid bool) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: valid}
}

func TestAttemptsRemaining(t *testing.T) {
	cases := []struct {
		failed int32
		want   int
	}{
		{failed: 0, want: 5},
		{failed: 1, want: 4},
		{failed: 4, want: 1},
		{failed: 5, want: 0},
		{failed: 9, want: 0},
	}
	for _, tc := range cases {
		if got := auth.AttemptsRemaining(tc.failed); got != tc.want {
			t.Errorf("AttemptsRemaining(%d) = %d, want %d", tc.failed, got, tc.want)
		}
	}
}

func TestLockedFor(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		until time.Time
		valid bool
		want  time.Duration
	}{
		{name: "never locked", until: time.Time{}, valid: false, want: 0},
		{name: "lock still running", until: now.Add(4 * time.Minute), valid: true, want: 4 * time.Minute},
		{name: "lock just expired", until: now, valid: true, want: 0},
		{name: "lock long expired", until: now.Add(-time.Hour), valid: true, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := auth.LockedFor(pgTime(tc.until, tc.valid), now); got != tc.want {
				t.Errorf("LockedFor(%v, %v) = %v, want %v", tc.until, now, got, tc.want)
			}
		})
	}
}

// TestLockedForRoundsPartialMinutesUp keeps the message honest: telling
// someone to come back "in 1 minute" when 90 seconds are left sends them
// back into a locked form.
func TestLockedForRoundsPartialMinutesUp(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		left time.Duration
		want int
	}{
		{left: time.Second, want: 1},
		{left: 60 * time.Second, want: 1},
		{left: 61 * time.Second, want: 2},
		{left: 15 * time.Minute, want: 15},
	}
	for _, tc := range cases {
		got := auth.LockMinutes(auth.LockedFor(pgTime(now.Add(tc.left), true), now))
		if got != tc.want {
			t.Errorf("LockMinutes(%v) = %d, want %d", tc.left, got, tc.want)
		}
	}
}
