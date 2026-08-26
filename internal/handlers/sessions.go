package handlers

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// sessionTimestamp formats a session's created_at in Vietnam local time,
// matching the "11 Aug 2026" convention the rest of the app uses for dates
// that need a year (a session can outlive the month it started in, unlike
// the transaction rows dateShort formats).
func sessionTimestamp(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(vietnamLocation).Format("02 Jan 2006, 15:04")
}

// deviceLabel turns a raw User-Agent string into something a person can
// recognize at a glance in the session list ("Chrome on Windows") rather
// than the full UA string. It only recognizes the handful of browser/OS
// pairs common enough to be worth naming; anything else falls back to the
// raw string, since a wrong guess is worse than no guess.
func deviceLabel(userAgent string) string {
	if userAgent == "" {
		return "Unknown device"
	}

	os := detectOS(userAgent)
	browser := detectBrowser(userAgent)
	if os == "" || browser == "" {
		return userAgent
	}
	return browser + " on " + os
}

func detectOS(ua string) string {
	switch {
	case strings.Contains(ua, "iPhone"):
		return "iPhone"
	case strings.Contains(ua, "iPad"):
		return "iPad"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Macintosh"):
		return "macOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	default:
		return ""
	}
}

func detectBrowser(ua string) string {
	switch {
	// Edge and Chrome both carry a "Chrome/" token, so the more specific
	// match has to run first.
	case strings.Contains(ua, "Edg/"):
		return "Edge"
	case strings.Contains(ua, "Chrome/"):
		return "Chrome"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	// Chrome's UA also contains "Safari/", so Safari only matches once
	// Chrome and Edge have already been ruled out above.
	case strings.Contains(ua, "Safari/"):
		return "Safari"
	default:
		return ""
	}
}
