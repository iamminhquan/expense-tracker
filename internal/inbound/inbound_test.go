package inbound_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"expensetracker/internal/inbound"
)

func TestSignAndVerifyRoundTrip(t *testing.T) {
	body := []byte(`{"from":"a@b.c"}`)
	sig := inbound.Sign("s3cret", body)
	if !inbound.Verify("s3cret", body, sig) {
		t.Errorf("Verify(secret, body, Sign(...)) = false, want true")
	}
}

func TestVerifyRejectsWrongSecretBodyOrSignature(t *testing.T) {
	body := []byte(`{"from":"a@b.c"}`)
	sig := inbound.Sign("s3cret", body)

	cases := []struct {
		name           string
		secret, sigArg string
		body           []byte
	}{
		{"wrong secret", "other", sig, body},
		{"tampered body", "s3cret", sig, []byte(`{"from":"evil@x.y"}`)},
		{"garbage signature", "s3cret", "not-hex", body},
		{"empty signature", "s3cret", "", body},
		{"empty secret", "", sig, body},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if inbound.Verify(tc.secret, tc.body, tc.sigArg) {
				t.Errorf("Verify(%q, %q, %q) = true, want false", tc.secret, tc.body, tc.sigArg)
			}
		})
	}
}

func TestSignIsStableAcrossCalls(t *testing.T) {
	body := []byte("abc")
	if a, b := inbound.Sign("k", body), inbound.Sign("k", body); a != b {
		t.Errorf("Sign twice = %q, %q, want equal", a, b)
	}
}

func TestParsePayloadReadsTheWorkerShape(t *testing.T) {
	raw := []byte(`{"from":"no-reply@mb.com.vn","to":"abc123@in.example.site",
		"subject":"Bien dong so du","messageId":"<m1@mail>","text":"body here"}`)
	got, err := inbound.ParsePayload(raw)
	if err != nil {
		t.Fatalf("ParsePayload() error = %v, want nil", err)
	}
	if got.From != "no-reply@mb.com.vn" {
		t.Errorf("From = %q, want %q", got.From, "no-reply@mb.com.vn")
	}
	if got.MessageID != "<m1@mail>" {
		t.Errorf("MessageID = %q, want %q", got.MessageID, "<m1@mail>")
	}
	if got.Text != "body here" {
		t.Errorf("Text = %q, want %q", got.Text, "body here")
	}
}

func TestParsePayloadRejectsEmptyEnvelope(t *testing.T) {
	if _, err := inbound.ParsePayload([]byte(`{"from":"","text":""}`)); err == nil {
		t.Error("ParsePayload(empty envelope) error = nil, want error")
	}
}

// TestSignMatchesTheJSSide pins the exact HMAC-SHA256 hex Sign produces for a
// known secret and body. emailworker/test/sign.test.mjs asserts this same
// literal for the JS implementation -- changing this constant without
// changing that one silently breaks ingestion, because the Worker and this
// package have to agree on every byte they sign.
func TestSignMatchesTheJSSide(t *testing.T) {
	const want = "adbde1ce40c89c14215687d5d762a47df6dfaefcfad61e2e86718ffc8498571b"
	if got := inbound.Sign("s3cret", []byte("{}")); got != want {
		t.Errorf("Sign(%q, %q) = %q, want %q", "s3cret", "{}", got, want)
	}
}

func TestNewTokenIsLongAndUnique(t *testing.T) {
	a, err := inbound.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v, want nil", err)
	}
	b, _ := inbound.NewToken()
	if a == b {
		t.Errorf("NewToken() twice = %q both times, want different", a)
	}
	if len(a) < 40 {
		t.Errorf("len(NewToken()) = %d, want >= 40", len(a))
	}
	if a != strings.ToLower(a) {
		t.Errorf("NewToken() = %q, want lowercase (mail systems normalize local parts)", a)
	}
}

func TestFingerprintDependsOnEveryField(t *testing.T) {
	base := inbound.Payload{From: "a@b.c", Subject: "s", Text: "t"}
	same := inbound.Fingerprint(base)
	if same != inbound.Fingerprint(base) {
		t.Error("Fingerprint is not stable for the same payload")
	}
	// Test that changing any field produces a different fingerprint.
	for _, changed := range []inbound.Payload{
		{From: "z@b.c", Subject: "s", Text: "t"},
		{From: "a@b.c", Subject: "z", Text: "t"},
		{From: "a@b.c", Subject: "s", Text: "z"},
	} {
		if inbound.Fingerprint(changed) == same {
			t.Errorf("Fingerprint(%+v) = same as base, want different", changed)
		}
	}
	// Test that boundary shifts (length-prefix misalignment) produce
	// different fingerprints. {From: "ab", Subject: "c"} must differ from
	// {From: "a", Subject: "bc"}, which would collide without length prefixes.
	fp1 := inbound.Fingerprint(inbound.Payload{From: "ab", Subject: "c"})
	fp2 := inbound.Fingerprint(inbound.Payload{From: "a", Subject: "bc"})
	if fp1 == fp2 {
		t.Errorf("Fingerprint with shifted boundaries = same, want different (length prefixes required)")
	}
}

func TestTruncateCapsAtMaxBodyBytes(t *testing.T) {
	long := strings.Repeat("x", inbound.MaxBodyBytes+500)
	if got := len(inbound.Truncate(long, inbound.MaxBodyBytes)); got != inbound.MaxBodyBytes {
		t.Errorf("len(Truncate(long, MaxBodyBytes)) = %d, want %d", got, inbound.MaxBodyBytes)
	}
	if got := inbound.Truncate("short", inbound.MaxBodyBytes); got != "short" {
		t.Errorf("Truncate(%q, MaxBodyBytes) = %q, want unchanged", "short", got)
	}
}

func TestTruncateRespectsUTF8Boundaries(t *testing.T) {
	// Create a string with multi-byte UTF-8 runes that exceeds MaxBodyBytes.
	// Vietnamese text uses multi-byte runes; a naive byte-slice would split them.
	multibyte := strings.Repeat("ế", inbound.MaxBodyBytes/3+100) // "ế" is 3 bytes
	got := inbound.Truncate(multibyte, inbound.MaxBodyBytes)
	if !utf8.ValidString(got) {
		t.Errorf("Truncate(multibyte, MaxBodyBytes) produced invalid UTF-8")
	}
	if len(got) > inbound.MaxBodyBytes {
		t.Errorf("len(Truncate(multibyte, MaxBodyBytes)) = %d, want <= %d", len(got), inbound.MaxBodyBytes)
	}
}
