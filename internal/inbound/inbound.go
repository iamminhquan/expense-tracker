// Package inbound holds the rules the email ingestion path shares with the
// Cloudflare Email Worker in emailworker/: the payload the Worker sends, the
// signature it stamps on it, and the token that names which account a
// forwarded message belongs to.
//
// Nothing here touches a request or the database, which is what lets every
// one of those rules be tested without Postgres -- the same split
// internal/csvimport draws between reading a file and writing rows.
package inbound

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxBodyBytes caps the stored plain-text body. Bank notices are a few KB;
// the cap exists so one oversized forward cannot fill the free-tier database.
const MaxBodyBytes = 64 * 1024

// MaxRawBytes caps the original MIME message kept beside the extracted text.
// It is larger than MaxBodyBytes because the raw form carries headers, markup
// and transfer encoding around the same content. Its purpose is replay: the
// extraction step has been wrong before -- an HTML-only notice whose entities
// were left undecoded -- and without the original there is nothing to run a
// fixed extractor against.
const MaxRawBytes = 2 * MaxBodyBytes

// SignatureHeader carries the hex HMAC the Worker stamps on the request body.
const SignatureHeader = "X-Inbox-Signature"

// ErrEmptyPayload is returned when a payload carries neither a sender nor a
// body, which means the Worker sent something that cannot be a bank notice.
var ErrEmptyPayload = errors.New("payload has no sender or body")

// Payload is the JSON the Email Worker POSTs. The field names are the
// contract: change one here and emailworker/src/index.js changes with it, in
// the same commit, or email silently stops arriving.
type Payload struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	MessageID string `json:"messageId"`
	Text      string `json:"text"`
	// Raw is the original MIME message, kept so a later fix to the text
	// extraction can be replayed against what actually arrived.
	Raw string `json:"raw"`
}

// ParsePayload decodes the Worker's JSON body.
func ParsePayload(b []byte) (Payload, error) {
	var p Payload
	if err := json.Unmarshal(b, &p); err != nil {
		return Payload{}, fmt.Errorf("decode payload: %w", err)
	}
	if strings.TrimSpace(p.From) == "" && strings.TrimSpace(p.Text) == "" {
		return Payload{}, ErrEmptyPayload
	}
	return p, nil
}

// Sign returns the hex HMAC-SHA256 of body under secret.
//
// The signature covers the body alone and carries no timestamp: replaying a
// captured request cannot create a second row, because bank_emails has a
// unique index on (user_id, message_id) and a replay carries the same
// Message-ID it did the first time.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether signature is Sign(secret, body). An empty secret
// never verifies: a deployment that forgot to configure one must reject every
// request rather than accept every request.
func Verify(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	want := Sign(secret, body)
	return subtle.ConstantTimeCompare([]byte(want), []byte(signature)) == 1
}

// NewToken returns the random local part of an account's inbox address: 32
// bytes from crypto/rand, base32 lowercase with no padding. The alphabet
// is case-insensitive because mail systems normalize the local part of an
// address to lowercase (Gmail does), so any tokens containing uppercase
// letters would be unreachable after a real email round-trip.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}

// Fingerprint is the dedupe key for a message whose Message-ID header is
// missing -- not every forwarder sets one, and the unique index still needs a
// value. Same idea as csvimport's Import.Fingerprint: a digest of what was
// actually read.
func Fingerprint(p Payload) string {
	h := sha256.New()
	// Length-prefixed so that moving text between fields cannot collide.
	for _, part := range []string{p.From, p.Subject, p.Text} {
		fmt.Fprintf(h, "%d:%s", len(part), part)
	}
	return "fp-" + hex.EncodeToString(h.Sum(nil))
}

// TruncateBody caps the extracted plain text at MaxBodyBytes.
func TruncateBody(s string) string {
	return truncateUTF8(s, MaxBodyBytes)
}

// TruncateRaw caps the original MIME message at MaxRawBytes, on the same rune
// boundary rule as TruncateBody: a message cut through a multi-byte character
// is invalid UTF-8, which Postgres refuses outright.
func TruncateRaw(s string) string {
	return truncateUTF8(s, MaxRawBytes)
}

// truncateUTF8 cuts s to at most max bytes without splitting a multi-byte
// character. Vietnamese bank text is full of multi-byte runes, and a string cut
// through one is invalid UTF-8 that Postgres refuses outright -- which would
// lose the whole message rather than shorten it.
func truncateUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	truncated := s[:max]
	// Drop trailing bytes while the last rune decodes as utf8.RuneError with
	// size 1, which is what a severed multi-byte sequence looks like. A real
	// U+FFFD decodes with size 3, so this cannot eat a legitimate one.
	for len(truncated) > 0 {
		r, size := utf8.DecodeLastRuneInString(truncated)
		if r != utf8.RuneError || size != 1 {
			break
		}
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}
