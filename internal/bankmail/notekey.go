package bankmail

import "strings"

// diacriticFold maps every lowercase Vietnamese letter that carries a tone
// mark or vowel-quality diacritic to its bare Latin letter, đ included.
// NoteKey needs it because two spellings of the same transfer note that
// differ only in diacritics must still fold to one lookup key.
var diacriticFold = map[rune]rune{
	'à': 'a', 'á': 'a', 'ả': 'a', 'ã': 'a', 'ạ': 'a',
	'ă': 'a', 'ằ': 'a', 'ắ': 'a', 'ẳ': 'a', 'ẵ': 'a', 'ặ': 'a',
	'â': 'a', 'ầ': 'a', 'ấ': 'a', 'ẩ': 'a', 'ẫ': 'a', 'ậ': 'a',
	'è': 'e', 'é': 'e', 'ẻ': 'e', 'ẽ': 'e', 'ẹ': 'e',
	'ê': 'e', 'ề': 'e', 'ế': 'e', 'ể': 'e', 'ễ': 'e', 'ệ': 'e',
	'ì': 'i', 'í': 'i', 'ỉ': 'i', 'ĩ': 'i', 'ị': 'i',
	'ò': 'o', 'ó': 'o', 'ỏ': 'o', 'õ': 'o', 'ọ': 'o',
	'ô': 'o', 'ồ': 'o', 'ố': 'o', 'ổ': 'o', 'ỗ': 'o', 'ộ': 'o',
	'ơ': 'o', 'ờ': 'o', 'ớ': 'o', 'ở': 'o', 'ỡ': 'o', 'ợ': 'o',
	'ù': 'u', 'ú': 'u', 'ủ': 'u', 'ũ': 'u', 'ụ': 'u',
	'ư': 'u', 'ừ': 'u', 'ứ': 'u', 'ử': 'u', 'ữ': 'u', 'ự': 'u',
	'ỳ': 'y', 'ý': 'y', 'ỷ': 'y', 'ỹ': 'y', 'ỵ': 'y',
	'đ': 'd',
}

// foldDiacritics removes tone marks and vowel-quality diacritics from s,
// folding đ to d and leaving every other rune -- including digits and plain
// Latin letters -- untouched. s must already be lowercase; diacriticFold
// only carries lowercase entries.
func foldDiacritics(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if folded, ok := diacriticFold[r]; ok {
			r = folded
		}
		b.WriteRune(r)
	}
	return b.String()
}

// NoteKey normalizes a bank transfer note into a lookup key for the
// category-hint memory a later slice builds. It is exported because two
// call sites must agree on exactly the same rule -- processing an incoming
// email, and a user correcting a transaction's category -- and two subtly
// different rules would leave hints that never match.
//
// A transfer note almost always carries a reference code unique to that one
// transaction (an FT number, a bank's own MBVCB-style stamp), so any token
// containing at least one digit is dropped, not just a token that is purely
// numeric. Keeping it would mean every transaction mints a brand new key and
// the hint memory never matches twice -- this is the rule the whole design
// depends on.
func NoteKey(description string) string {
	folded := foldDiacritics(strings.ToLower(description))
	fields := strings.Fields(folded)
	kept := fields[:0]
	for _, tok := range fields {
		if strings.ContainsAny(tok, "0123456789") {
			continue
		}
		kept = append(kept, tok)
	}
	return strings.Join(kept, " ")
}
