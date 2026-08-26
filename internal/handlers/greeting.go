package handlers

import "time"

// greeting is the salutation the dashboard heading opens with, chosen from
// the hour on t's own clock. The bands are deliberately uneven: the small
// hours get their own line rather than being folded into "Good night",
// because someone opening a spending tracker at 3am is not being wished a
// good night.
//
// The caller supplies the location -- see greetingLine's callers, which read
// the hour in Vietnam time rather than the server's.
func greeting(t time.Time) string {
	switch h := t.Hour(); {
	case h >= 5 && h < 11:
		return "Good morning"
	case h >= 11 && h < 17:
		return "Good afternoon"
	case h >= 17 && h < 22:
		return "Good evening"
	case h >= 22 || h < 1:
		return "Good night"
	default:
		return "Night owl"
	}
}

// greetingLine is the finished heading: the salutation, then the user's name
// if there is one. An account with no name drops the comma along with it,
// rather than trailing one in front of nothing.
func greetingLine(t time.Time, name string) string {
	if name == "" {
		return greeting(t)
	}
	return greeting(t) + ", " + name
}
