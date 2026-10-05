package pgval_test

import (
	"testing"

	"expensetracker/internal/pgval"
)

// The one thing here that is a decision rather than a struct literal: an
// empty string is a present value, not NULL. A column set from Text("")
// holds ”, which is what every "no note" transaction stores.
func TestTextKeepsAnEmptyStringPresent(t *testing.T) {
	got := pgval.Text("")
	if !got.Valid || got.String != "" {
		t.Errorf("Text(%q) = %+v, want {String:\"\" Valid:true}", "", got)
	}
}
