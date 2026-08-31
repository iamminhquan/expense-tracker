package pgval_test

import (
	"testing"
	"time"

	"expensetracker/internal/pgval"
)

func TestInt64IsAlwaysPresent(t *testing.T) {
	// Zero is a real id-shaped value, not an absent one: only the caller
	// choosing not to call these ever means NULL.
	got := pgval.Int64(0)
	if !got.Valid || got.Int64 != 0 {
		t.Errorf("Int64(0) = %+v, want {Int64:0 Valid:true}", got)
	}
}

func TestTextKeepsAnEmptyStringPresent(t *testing.T) {
	got := pgval.Text("")
	if !got.Valid || got.String != "" {
		t.Errorf("Text(%q) = %+v, want {String:\"\" Valid:true}", "", got)
	}
}

func TestDateCarriesTheTimeThrough(t *testing.T) {
	when := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	got := pgval.Date(when)
	if !got.Valid || !got.Time.Equal(when) {
		t.Errorf("Date(%s) = %+v, want that time, Valid:true", when, got)
	}
}
