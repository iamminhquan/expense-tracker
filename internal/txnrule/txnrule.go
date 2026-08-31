// Package txnrule holds the limits a transaction has to satisfy whichever
// way it enters the app -- the quick-add form, the inline edit, or a CSV
// file. The numbers lived in three places before this package, one of them
// with a comment promising it mirrored the other two; a rule the importer
// spelled more leniently than the form would be a second, laxer way in.
//
// Only the numbers and the predicates live here. The wording of a failure
// stays with whoever shows it: the form retargets a sentence at a field,
// while the importer names the line and the value it read.
package txnrule

import "time"

// MaxNoteRunes is how long a transaction note may be, counted in runes
// rather than bytes so a Vietnamese note is measured the way it reads.
const MaxNoteRunes = 200

// MaxFutureDays is how far ahead a transaction may be dated. A little
// slack is deliberate -- a bill paid tomorrow is a real thing to record --
// but a ledger of the future is not what this app is.
const MaxFutureDays = 7

// NoteTooLong reports whether note exceeds MaxNoteRunes.
func NoteTooLong(note string) bool {
	return len([]rune(note)) > MaxNoteRunes
}

// TooFarInFuture reports whether occurredOn is dated more than
// MaxFutureDays past now. Both times are compared as the caller hands
// them over: the form passes Vietnam-local time, since that is the day the
// user believes it is, and the importer passes whatever clock it was given.
func TooFarInFuture(occurredOn, now time.Time) bool {
	return occurredOn.After(now.AddDate(0, 0, MaxFutureDays))
}
