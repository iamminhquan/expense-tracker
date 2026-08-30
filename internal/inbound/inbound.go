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

// TruncateBody caps a body at MaxBodyBytes while respecting UTF-8 boundaries.
// A slice by byte count alone can split a multi-byte rune, producing invalid
// UTF-8 that later queries cannot store. This matters for Vietnamese text,
// which is full of multi-byte runes.
func TruncateBody(s string) string {
	if len(s) <= MaxBodyBytes {
		return s
	}
	truncated := s[:MaxBodyBytes]
	// Trim back to the last complete rune by dropping bytes while the last
	// rune decodes as utf8.RuneError.
	for len(truncated) > 0 {
		r, size := utf8.DecodeLastRuneInString(truncated)
		if r != utf8.RuneError || size != 1 {
			break
		}
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}
