package auth

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// The login throttle: MaxLoginAttempts consecutive wrong passwords lock an
// account for LockoutWindow, after which it unlocks on its own. Both live
// here rather than in the handler because the SQL that stamps the lock and
// the message that explains it have to agree on the same numbers.
const (
	MaxLoginAttempts = 5
	LockoutWindow    = 15 * time.Minute

	// WarnAtRemaining is when the sign-in form starts counting down out
	// loud. Warning from the first wrong password would mostly nag people
	// who simply mistyped.
	WarnAtRemaining = 2
)

// AttemptsRemaining reports how many wrong passwords an account has left
// before it locks, never going below zero.
func AttemptsRemaining(failed int32) int {
	left := MaxLoginAttempts - int(failed)
	if left < 0 {
		return 0
	}
	return left
}

// LockedFor reports how much of a lock is left at now, or zero if the
// account is not locked. A lapsed timestamp is not cleared anywhere -- it
// simply stops counting -- so "expired" and "never locked" both land here
// as zero.
func LockedFor(lockedUntil pgtype.Timestamptz, now time.Time) time.Duration {
	if !lockedUntil.Valid {
		return 0
	}
	left := lockedUntil.Time.Sub(now)
	if left <= 0 {
		return 0
	}
	return left
}

// LockMinutes rounds a remaining lock up to whole minutes for display,
// because telling someone to return in 1 minute when 90 seconds are left
// walks them straight back into a locked form.
func LockMinutes(left time.Duration) int {
	return int((left + time.Minute - 1) / time.Minute)
}
