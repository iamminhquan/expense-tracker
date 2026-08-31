package mailer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/mailer"
)

func TestConfigured(t *testing.T) {
	cases := []struct {
		name string
		cfg  mailer.Config
		want bool
	}{
		{"empty", mailer.Config{}, false},
		{"key only", mailer.Config{APIKey: "k"}, false},
		{"from only", mailer.Config{From: "a@b.com"}, false},
		{"both", mailer.Config{APIKey: "k", From: "a@b.com"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mailer.New(tc.cfg).Configured(); got != tc.want {
				t.Errorf("Configured() with %+v = %v, want %v", tc.cfg, got, tc.want)
			}
		})
	}
}

// TestSendPostsExpectedRequest can't hit Brevo's real endpoint, so it
// swaps in an httptest server as a stand-in to pin down the one thing that
// would otherwise only surface against the live API: the request shape
// (method, headers, JSON body) Send builds.
func TestSendPostsExpectedRequest(t *testing.T) {
	var gotMethod, gotAPIKey, gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAPIKey = r.Header.Get("api-key")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	m := mailer.New(mailer.Config{APIKey: "test-key", From: "sender@example.com", Endpoint: srv.URL})
	if err := m.Send(context.Background(), "to@example.com", "Subject line", "body text"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotAPIKey != "test-key" {
		t.Errorf("api-key header = %q, want %q", gotAPIKey, "test-key")
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if got := gotBody["subject"]; got != "Subject line" {
		t.Errorf("body[subject] = %v, want %q", got, "Subject line")
	}
	if got := gotBody["textContent"]; got != "body text" {
		t.Errorf("body[textContent] = %v, want %q", got, "body text")
	}
	sender, _ := gotBody["sender"].(map[string]any)
	if sender["email"] != "sender@example.com" {
		t.Errorf("body[sender][email] = %v, want %q", sender["email"], "sender@example.com")
	}
	to, _ := gotBody["to"].([]any)
	if len(to) != 1 {
		t.Fatalf("body[to] = %v, want one recipient", to)
	}
	firstTo, _ := to[0].(map[string]any)
	if firstTo["email"] != "to@example.com" {
		t.Errorf("body[to][0][email] = %v, want %q", firstTo["email"], "to@example.com")
	}
}

func TestSendReturnsErrorOnNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer srv.Close()

	m := mailer.New(mailer.Config{APIKey: "bad-key", From: "sender@example.com", Endpoint: srv.URL})
	err := m.Send(context.Background(), "to@example.com", "Subject", "body")
	if err == nil {
		t.Fatal("Send() error = nil, want an error on a 401 response")
	}
}
