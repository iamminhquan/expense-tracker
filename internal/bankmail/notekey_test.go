package bankmail_test

import (
	"testing"

	"expensetracker/internal/bankmail"
)

func TestNoteKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"lowercases", "NGUYEN VAN A chuyen tien", "nguyen van a chuyen tien"},
		{
			"same key across differing reference codes, FT prefix",
			"NGUYEN VAN A chuyen tien FT24123456789",
			"nguyen van a chuyen tien",
		},
		{
			"same key across differing reference codes, MBVCB prefix",
			"NGUYEN VAN A chuyen tien MBVCB.9876543210",
			"nguyen van a chuyen tien",
		},
		{"strips a bare numeric token", "Thanh toan GRAB 8829471", "thanh toan grab"},
		{"strips Vietnamese diacritics", "Chuyển tiền học phí", "chuyen tien hoc phi"},
		{
			// đ is not an accented Latin d -- it is a distinct Vietnamese
			// letter (U+0111), so a "strip combining marks" implementation
			// would leave it untouched. This pins that it must fold to d.
			"folds đ to d",
			"Chuyen tien mua đồ ăn",
			"chuyen tien mua do an",
		},
		{"empty string stays empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := bankmail.NoteKey(tc.in); got != tc.want {
				t.Errorf("NoteKey(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
