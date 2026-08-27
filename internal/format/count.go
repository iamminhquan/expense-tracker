package format

import "strconv"

// CountOf renders a count with its noun in agreement: "1 transaction",
// "2 transactions". English needs this and Vietnamese does not, so the
// templates carried a bare plural noun until the UI was translated.
//
// The count arrives as an int from len() in one template and as an int64
// straight out of a COUNT(*) in another, so it is taken as any rather than
// forcing one of the two call sites to convert in the template.
func CountOf(n any, singular string) string {
	var count int64
	switch v := n.(type) {
	case int:
		count = int64(v)
	case int64:
		count = v
	case int32:
		count = int64(v)
	}
	if count == 1 {
		return "1 " + singular
	}
	return strconv.FormatInt(count, 10) + " " + singular + "s"
}
