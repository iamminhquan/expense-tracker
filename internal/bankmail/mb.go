package bankmail

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// mbLabels are every field label a real MB eBanking transfer notice carries.
// They are listed in no particular order: extractFields locates each one
// wherever it actually sits in the body, since a notice missing a field
// (no debit account, say) still has every other label right where the
// template usually puts it, just with its neighbor's value filling the gap.
var mbLabels = []string{
	"Ngày, giờ giao dịch",
	"Loại giao dịch",
	"Số tham chiếu",
	"Tài khoản trích nợ",
	"Người thụ hưởng",
	"Số tiền giao dịch",
	"Nội dung chuyển tiền",
	"Cách thức lệnh",
	"Tình trạng",
}

// mbSuccessStatus is the only "Tình trạng" value this package treats as a
// real transaction. A failed or pending transfer never moved money, so
// recording one would put a transaction in the ledger that never happened.
const mbSuccessStatus = "Giao dịch thành công"

var collapseWhitespace = regexp.MustCompile(`\s+`)

// parseMB reads one MB eBanking "Chuyển tiền nội bộ MB" transfer notice --
// the only MB shape this app has a real sample of. An unrecognized shape
// (an OTP message, an ad, a failed transfer, a field missing) returns
// ErrNotANotice rather than a guess: see this plan's "Phạm vi bị thu hẹp vì
// thiếu mẫu" for why guessing here is worse than not parsing at all.
func parseMB(_, body string) (Notice, error) {
	// The source is an HTML table, so a label can be split across a
	// newline mid-phrase ("Số\n tiền giao dịch"). Collapsing every run of
	// whitespace (newlines included) to one space is what makes the label
	// match at all.
	collapsed := strings.TrimSpace(collapseWhitespace.ReplaceAllString(body, " "))
	fields := extractFields(collapsed, mbLabels)

	// Prefix, not equality: "Tình trạng" is the last label MB's template
	// emits, so extractFields has no following label to stop at and the
	// value runs to the end of the body -- carrying MB's whole footer
	// ("Xin chân thành cảm ơn. Hội sở: ...") along with it. Comparing for
	// equality here is what made every real notice fail this gate while
	// the fixture, captured with the footer trimmed off, passed.
	if !strings.HasPrefix(strings.TrimSpace(fields["Tình trạng"]), mbSuccessStatus) {
		return Notice{}, ErrNotANotice
	}

	// Direction is inferred from structure, never guessed: a debit account
	// present means money left the account. No sample of an incoming
	// transfer notice exists yet, so its absence is refused rather than
	// assumed to mean income.
	if _, hasDebitAccount := fields["Tài khoản trích nợ"]; !hasDebitAccount {
		return Notice{}, ErrNotANotice
	}

	amountText, ok := fields["Số tiền giao dịch"]
	if !ok {
		return Notice{}, ErrNotANotice
	}
	amount, err := parseMBAmount(amountText)
	if err != nil {
		return Notice{}, ErrNotANotice
	}

	whenText, ok := fields["Ngày, giờ giao dịch"]
	if !ok {
		return Notice{}, ErrNotANotice
	}
	when, err := time.ParseInLocation("02-01-2006 15:04:05", whenText, vietnamLocation)
	if err != nil {
		return Notice{}, ErrNotANotice
	}

	description, ok := fields["Nội dung chuyển tiền"]
	if !ok {
		return Notice{}, ErrNotANotice
	}

	// The notice also carries a post-transaction balance ("Số dư"-style
	// field on some MB templates), but Notice has no field for it and never
	// will: a value parsed only to be discarded is dead code.
	return Notice{
		Bank:               "mb",
		Amount:             amount,
		Direction:          "expense",
		OccurredAt:         when,
		Description:        description,
		DebitAccount:       accountNumber(fields["Tài khoản trích nợ"]),
		BeneficiaryAccount: accountNumber(fields["Người thụ hưởng"]),
	}, nil
}

// accountNumber pulls the account number out of a field MB writes as
// "NGUYEN VAN A - 0001111111111 (VND)": the longest run of digits in it.
//
// Longest-run rather than "everything after the last dash", because an
// account holder's name is free text that may itself contain a dash, while
// no MB account field has ever carried a second number long enough to
// outrun the account itself. An empty result means the field was absent or
// carried no digits, which every caller must treat as "unknown" rather than
// as a match -- comparing two empty strings would make every notice with a
// missing field look like a self-transfer.
func accountNumber(field string) string {
	var longest, current string
	for _, r := range field {
		if r >= '0' && r <= '9' {
			current += string(r)
			if len(current) > len(longest) {
				longest = current
			}
			continue
		}
		current = ""
	}
	return longest
}

// extractFields finds where each label in labels actually occurs in text
// and returns the text between it and whichever label comes next -- not the
// next label in the labels slice, but the next one actually present in the
// body, since a notice missing a field mid-template would otherwise let its
// neighbor's value swallow the gap left behind.
//
// Because the next boundary is "wherever the next label's exact text next
// occurs in the body," not a bound tied to any particular field, a value
// that happened to contain another label's exact text verbatim would cut
// short at that point and misattribute the rest. Not reachable with the one
// template this package supports today; worth remembering if a second bank
// or a second MB shape is ever added on top of this same extractor.
func extractFields(text string, labels []string) map[string]string {
	type hit struct {
		label string
		start int
		end   int
	}
	var hits []hit
	for _, label := range labels {
		idx := strings.Index(text, label)
		if idx < 0 {
			continue
		}
		hits = append(hits, hit{label: label, start: idx, end: idx + len(label)})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].start < hits[j].start })

	fields := make(map[string]string, len(hits))
	for i, h := range hits {
		valueEnd := len(text)
		if i+1 < len(hits) {
			valueEnd = hits[i+1].start
		}
		fields[h.label] = strings.TrimSpace(text[h.end:valueEnd])
	}
	return fields
}

// mbAmountShape is MB's one amount shape: an optional run of comma-grouped
// thousands (each group exactly three digits, per standard grouping -- "1"
// then ",234" then ",567", never a short or long group) followed by an
// optional single "." and one or two decimal digits. Anything else --
// "20.000,00" (the swapped, dot-thousands convention), two dots, a
// mis-sized group, three decimal digits -- fails this and is refused before
// a single comma is stripped, which is what makes the strictness real: a
// naive "strip commas, split on last dot" reading (what this function used
// to do) silently turns "20.000,00" into 20 instead of 20,000.
var mbAmountShape = regexp.MustCompile(`^\d{1,3}(,\d{3})*(\.\d{1,2})?$`)

// parseMBAmount reads MB's one amount format strictly rather than reusing
// csvimport.parseAmount's lenient rules: MB always writes "(VND) 20,000.00"
// -- comma as the thousands separator, dot as the decimal point, no other
// shape. This parser faces exactly one sender emitting exactly one format,
// so tolerance here buys no real-world compatibility -- it only buys a
// silently wrong amount the day MB changes its template, in place of the
// ErrNotANotice / failed row a human can see and replay.
func parseMBAmount(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "(VND)")
	s = strings.TrimSpace(s)

	if !mbAmountShape.MatchString(s) {
		return 0, fmt.Errorf("parse mb amount %q: does not match MB's comma-thousands, dot-decimal shape", raw)
	}
	s = strings.ReplaceAll(s, ",", "")

	dot := strings.Index(s, ".")
	if dot < 0 {
		whole, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse mb amount %q: %w", raw, err)
		}
		return whole, nil
	}

	whole, err := strconv.ParseInt(s[:dot], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse mb amount %q: %w", raw, err)
	}
	// mbAmountShape guarantees 1-2 digits follow the dot, so fracPart is
	// never empty here.
	fracPart := s[dot+1:]
	fracNum, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse mb amount %q: %w", raw, err)
	}
	// Round to whole đồng: integer comparison against half the
	// denominator avoids the float imprecision a naive "* 100" round would
	// risk on a currency value.
	denom := int64(1)
	for range fracPart {
		denom *= 10
	}
	if 2*fracNum >= denom {
		whole++
	}
	return whole, nil
}
