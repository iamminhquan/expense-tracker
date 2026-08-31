// Package mailer sends the transactional email the app needs (currently
// just the password-reset link) through Brevo's transactional email HTTP
// API. HTTPS rather than SMTP is deliberate: Render's free tier (where this
// app is deployed) blocks outbound traffic to every SMTP port (25, 465,
// 587), so SMTP -- to Brevo's relay or anyone else's -- cannot work there,
// while port 443 is never blocked.
package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const sendEndpoint = "https://api.brevo.com/v3/smtp/email"

// Config holds the Brevo API settings. Both fields are optional, the same
// way the rest of internal/config treats non-critical settings: no API key
// just means Send fails (logged by the caller, not fatal) rather than the
// app refusing to start.
type Config struct {
	APIKey string
	From   string
	// Endpoint overrides the Brevo API URL, so a test can point Send at an
	// httptest server instead of the real API. Empty means sendEndpoint.
	Endpoint string
}

// Mailer sends email through the Brevo account it was constructed with.
type Mailer struct {
	cfg      Config
	client   *http.Client
	endpoint string
}

// New constructs a Mailer that sends through the Brevo account in cfg.
func New(cfg Config) *Mailer {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = sendEndpoint
	}
	return &Mailer{cfg: cfg, client: &http.Client{}, endpoint: endpoint}
}

// Configured reports whether enough settings are present to attempt a send.
// Callers use this to log a clear "not configured" message instead of
// letting Send fail with an opaque 401 against an empty API key.
func (m *Mailer) Configured() bool {
	return m.cfg.APIKey != "" && m.cfg.From != ""
}

type sendRequest struct {
	Sender      contact   `json:"sender"`
	To          []contact `json:"to"`
	Subject     string    `json:"subject"`
	TextContent string    `json:"textContent"`
}

type contact struct {
	Email string `json:"email"`
}

// Send delivers a plain-text email via Brevo's v3 transactional email
// endpoint. ctx is the caller's to cancel or time out the HTTP call; the
// caller must not pass a request-scoped context that outlives the request
// itself if the send is meant to keep running in the background (see
// internal/handlers/auth_password_reset.go).
func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	reqBody, err := json.Marshal(sendRequest{
		Sender:      contact{Email: m.cfg.From},
		To:          []contact{{Email: to}},
		Subject:     subject,
		TextContent: body,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("api-key", m.cfg.APIKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("brevo: unexpected status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}
