package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/api"
)

func TestForgotAndResetPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, _, _, _ := registerTestAccount(t, deps, router)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/forgot-password", map[string]string{"email": email})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/forgot-password = %d %s, want 200", rec.Code, rec.Body.String())
	}

	// The mailer isn't configured in tests (newTestDeps uses mailer.New
	// with an empty Config), so the token exists but was never emailed --
	// read it straight out of the database, the same workaround the HTML
	// side's own reset-password tests use.
	var token string
	err := deps.DB.QueryRow(context.Background(),
		"SELECT token FROM password_reset_tokens pt JOIN users u ON u.id = pt.user_id WHERE u.email = $1 ORDER BY pt.expires_at DESC LIMIT 1",
		email,
	).Scan(&token)
	if err != nil {
		t.Fatalf("query reset token: %v", err)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reset-password?token="+token, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/reset-password?token=... = %d %s, want 200", rec.Code, rec.Body.String())
	}

	const newPassword = "a-brand-new-password"
	rec = doJSON(t, router, http.MethodPost, "/api/v1/reset-password", map[string]string{
		"token": token, "password": newPassword, "passwordConfirm": newPassword,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/reset-password = %d %s, want 200", rec.Code, rec.Body.String())
	}
	decodeAuthResponse(t, rec) // signs the visitor in on success

	rec = doJSON(t, router, http.MethodPost, "/api/v1/login", map[string]string{"email": email, "password": newPassword})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/login with the new password = %d %s, want 200", rec.Code, rec.Body.String())
	}
}

func TestResetPasswordRejectsInvalidToken(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reset-password?token=not-a-real-token", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/v1/reset-password (bad token) = %d, want 404", rec.Code)
	}
	decodeError(t, rec)

	rec = doJSON(t, router, http.MethodPost, "/api/v1/reset-password", map[string]string{
		"token": "not-a-real-token", "password": "whatever123", "passwordConfirm": "whatever123",
	})
	if rec.Code != http.StatusNotFound {
		t.Errorf("POST /api/v1/reset-password (bad token) = %d, want 404", rec.Code)
	}
	decodeError(t, rec)
}

func TestVerifyEmail(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, _, _, _ := registerTestAccount(t, deps, router)

	var token string
	err := deps.DB.QueryRow(context.Background(),
		"SELECT token FROM email_verification_tokens vt JOIN users u ON u.id = vt.user_id WHERE u.email = $1 ORDER BY vt.expires_at DESC LIMIT 1",
		email,
	).Scan(&token)
	if err != nil {
		t.Fatalf("query verification token (register should have queued one): %v", err)
	}

	rec := doJSON(t, router, http.MethodPost, "/api/v1/verify-email", map[string]string{"token": token})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/verify-email = %d %s, want 200", rec.Code, rec.Body.String())
	}
	result := decodeData[struct {
		Verified bool `json:"verified"`
		Conflict bool `json:"conflict"`
	}](t, rec)
	if !result.Verified {
		t.Error("Verified = false, want true")
	}
	if result.Conflict {
		t.Error("Conflict = true, want false")
	}
}

func TestVerifyEmailInvalidTokenIsNotAnError(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/verify-email", map[string]string{"token": "not-a-real-token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/verify-email (bad token) = %d %s, want 200 (mirrors handlers.verifyEmailPage's zero-value outcome)", rec.Code, rec.Body.String())
	}
	result := decodeData[struct {
		Verified bool `json:"verified"`
		Conflict bool `json:"conflict"`
	}](t, rec)
	if result.Verified || result.Conflict {
		t.Errorf("got %+v, want both false for an invalid token", result)
	}
}
