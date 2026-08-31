package bankmail_test

import (
	"errors"
	"testing"
	"time"

	"expensetracker/internal/bankmail"
)

// mbTransferNotice is a real MB eBanking notice, account numbers masked.
// Its labels are broken across newlines ("Số\n tiền giao dịch") because the
// source was an HTML table -- that is exactly what a naive matcher fails on,
// and the reason Parse collapses whitespace before matching a label.
const mbTransferNotice = `Cảm ơn Quý khách đã sử dụng dịch vụ MB eBanking.

MB xin thông báo giao dịch của Quý khách đã được thực hiện như sau:



 Ngày,
 giờ giao dịch



 31-08-2026 01:05:33






 Loại
 giao dịch



 Chuyển tiền nội bộ MB






 Số
 tham chiếu



 26083101055223730






 Tài
 khoản trích nợ



 NGUYEN VAN A - 0001111111111 (VND)






 Người
 thụ hưởng



 Nguyen Van B - 0399999999






 Số
 tiền giao dịch



 (VND) 20,000.00






 Nội
 dung chuyển tiền



 NGUYEN VAN A chuyen tien






 Cách
 thức lệnh



 Thanh toán ngay






 Tình
 trạng



 Giao dịch thành công


Xin chân thành cảm ơn.

Hội sở: Toà nhà MB, Số 18 Lê Văn Lương – Cầu Giấy – Hà Nội – Việt Nam
ĐT: (+84) 24 6277 7222 | Fax: (+84) 24 6266 1080
Website: http://www.mbbank.com.vn
Liên hệ MB 247: 1900 545426 | (+84) 24 3767 4050
`

func mustVietnamTime(t *testing.T, layout, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*60*60)
	}
	got, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		t.Fatalf("ParseInLocation(%q, %q) error: %v", layout, value, err)
	}
	return got
}

func TestParseMBTransferNotice(t *testing.T) {
	n, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich thanh cong", mbTransferNotice)
	if err != nil {
		t.Fatalf("Parse(mbTransferNotice) unexpected error: %v", err)
	}
	if n.Bank != "mb" {
		t.Errorf("Parse(mbTransferNotice).Bank = %q, want %q", n.Bank, "mb")
	}
	if n.Amount != 20000 {
		t.Errorf("Parse(mbTransferNotice).Amount = %d, want %d", n.Amount, 20000)
	}
	if n.Direction != "expense" {
		t.Errorf("Parse(mbTransferNotice).Direction = %q, want %q", n.Direction, "expense")
	}
	wantTime := mustVietnamTime(t, "02-01-2006 15:04:05", "31-08-2026 01:05:33")
	if !n.OccurredAt.Equal(wantTime) {
		t.Errorf("Parse(mbTransferNotice).OccurredAt = %v, want %v", n.OccurredAt, wantTime)
	}
	if n.Description != "NGUYEN VAN A chuyen tien" {
		t.Errorf("Parse(mbTransferNotice).Description = %q, want %q", n.Description, "NGUYEN VAN A chuyen tien")
	}
}

func TestParseUnknownSender(t *testing.T) {
	_, err := bankmail.Parse("someone@example.com", "Thong bao giao dich thanh cong", mbTransferNotice)
	if !errors.Is(err, bankmail.ErrUnknownSender) {
		t.Errorf("Parse(unknown sender) error = %v, want errors.Is ErrUnknownSender", err)
	}
}

func TestParseMBAdvertisingBodyHasNoLabels(t *testing.T) {
	body := "Cam on quy khach da su dung dich vu MB eBanking. Uu dai thang nay danh cho ban!"
	_, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Uu dai thang 8", body)
	if !errors.Is(err, bankmail.ErrNotANotice) {
		t.Errorf("Parse(advertising body) error = %v, want errors.Is ErrNotANotice", err)
	}
}

func TestParseMBFailedTransactionStatus(t *testing.T) {
	body := `
 Ngày,
 giờ giao dịch

 31-08-2026 01:05:33

 Tài
 khoản trích nợ

 NGUYEN VAN A - 0001111111111 (VND)

 Số
 tiền giao dịch

 (VND) 20,000.00

 Nội
 dung chuyển tiền

 NGUYEN VAN A chuyen tien

 Tình
 trạng

 Giao dịch không thành công

Xin chân thành cảm ơn.

Hội sở: Toà nhà MB, Số 18 Lê Văn Lương – Cầu Giấy – Hà Nội – Việt Nam
Website: http://www.mbbank.com.vn
`
	// The footer matters here as much as it does in the success fixture:
	// the status gate matches a prefix precisely because the footer runs
	// on past the value, and this is what proves that relaxation still
	// refuses a transfer that never moved money.
	_, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich", body)
	if !errors.Is(err, bankmail.ErrNotANotice) {
		t.Errorf("Parse(failed transaction status) error = %v, want errors.Is ErrNotANotice", err)
	}
}

func TestParseMBMissingDebitAccount(t *testing.T) {
	body := `
 Ngày,
 giờ giao dịch

 31-08-2026 01:05:33

 Số
 tiền giao dịch

 (VND) 20,000.00

 Nội
 dung chuyển tiền

 NGUYEN VAN A chuyen tien

 Tình
 trạng

 Giao dịch thành công
`
	_, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich", body)
	if !errors.Is(err, bankmail.ErrNotANotice) {
		t.Errorf("Parse(missing debit account) error = %v, want errors.Is ErrNotANotice", err)
	}
}

// mbBodyWithAmount builds a minimal, otherwise-valid MB notice body with the
// given "Số tiền giao dịch" value, so every amount-shape case below exercises
// the same Parse path a real notice would, not parseMBAmount in isolation.
func mbBodyWithAmount(amount string) string {
	return `
 Ngày,
 giờ giao dịch

 31-08-2026 01:05:33

 Tài
 khoản trích nợ

 NGUYEN VAN A - 0001111111111 (VND)

 Số
 tiền giao dịch

 ` + amount + `

 Nội
 dung chuyển tiền

 NGUYEN VAN A chuyen tien

 Tình
 trạng

 Giao dịch thành công
`
}

func TestParseMBAmountValidShapes(t *testing.T) {
	tests := []struct {
		name   string
		amount string
		want   int64
	}{
		{"rounds a fraction at or past the half đồng up", "(VND) 1,234,567.89", 1234568},
		{"three-digit whole with two-digit decimal", "(VND) 500.00", 500},
		{"comma-grouped thousands with no decimal part at all", "(VND) 1,000", 1000},
		{"the real fixture's own amount", "(VND) 20,000.00", 20000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich", mbBodyWithAmount(tc.amount))
			if err != nil {
				t.Fatalf("Parse(amount %q) unexpected error: %v", tc.amount, err)
			}
			if n.Amount != tc.want {
				t.Errorf("Parse(amount %q).Amount = %d, want %d", tc.amount, n.Amount, tc.want)
			}
		})
	}
}

// TestParseMBAmountRejectsWrongShape guards the strictness parseMBAmount
// promises: an amount that does not match MB's one comma-thousands,
// dot-decimal shape must fail visibly (ErrNotANotice) rather than convert to
// a silently wrong number. "20.000,00" is the case that matters most here --
// the swapped, dot-thousands convention -- since stripping commas and
// splitting on the last dot without validating the shape first reads it as
// 20, a thousand-fold understatement of the real twenty thousand đồng.
func TestParseMBAmountRejectsWrongShape(t *testing.T) {
	tests := []struct {
		name   string
		amount string
	}{
		{"swapped dot-thousands, comma-decimal convention", "(VND) 20.000,00"},
		{"two dots, no thousands separator at all", "(VND) 20.000.00"},
		{"a thousands group that is not three digits", "(VND) 1,00,000.00"},
		{"a decimal part longer than two digits", "(VND) 20,000.123"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bankmail.Parse("mbebanking@mbbank.com.vn", "Thong bao giao dich", mbBodyWithAmount(tc.amount))
			if !errors.Is(err, bankmail.ErrNotANotice) {
				t.Errorf("Parse(amount %q) error = %v, want errors.Is ErrNotANotice", tc.amount, err)
			}
		})
	}
}
