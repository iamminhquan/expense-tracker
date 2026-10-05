package format_test

import (
	"testing"
	"time"

	"expensetracker/internal/format"
)

// at builds a time whose only significant part is the clock: Greeting reads
// nothing but the hour, and it is the caller that decides which location the
// hour is read in.
func at(hour, minute int) time.Time {
	return time.Date(2026, time.August, 26, hour, minute, 0, 0, time.UTC)
}

func TestGreeting(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{"midnight", at(0, 0), "Good night"},
		{"last minute of the night band", at(0, 59), "Good night"},
		{"night owl starts at 01:00", at(1, 0), "Night owl"},
		{"last minute before morning", at(4, 59), "Night owl"},
		{"morning starts at 05:00", at(5, 0), "Good morning"},
		{"last minute of morning", at(10, 59), "Good morning"},
		{"afternoon starts at 11:00", at(11, 0), "Good afternoon"},
		{"last minute of afternoon", at(16, 59), "Good afternoon"},
		{"evening starts at 17:00", at(17, 0), "Good evening"},
		{"last minute of evening", at(21, 59), "Good evening"},
		{"night starts at 22:00", at(22, 0), "Good night"},
		{"end of day", at(23, 59), "Good night"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := format.Greeting(tc.in); got != tc.want {
				t.Errorf("Greeting(%s) = %q, want %q", tc.in.Format("15:04"), got, tc.want)
			}
		})
	}
}

func TestGreetingLine(t *testing.T) {
	tests := []struct {
		name     string
		in       time.Time
		userName string
		want     string
	}{
		{"named", at(19, 30), "immq", "Good evening, immq"},
		{"no name leaves no dangling comma", at(19, 30), "", "Good evening"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := format.GreetingLine(tc.in, tc.userName); got != tc.want {
				t.Errorf("GreetingLine(%s, %q) = %q, want %q", tc.in.Format("15:04"), tc.userName, got, tc.want)
			}
		})
	}
}
