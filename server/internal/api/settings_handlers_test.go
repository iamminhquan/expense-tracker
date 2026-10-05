package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"expensetracker/internal/api"
)

// loginAndGetRefreshCookie exercises the real /api/v1/login flow (rather than
// authedRequest's direct access-token issuance) for the tests below that
// need a genuine refresh-token cookie to exercise session-revocation and
// "current session" identification.
func loginAndGetRefreshCookie(t *testing.T, deps api.Deps, router http.Handler, email, password string) (accessToken string, refreshCookie *http.Cookie) {
	t.Helper()
	rec := doJSON(t, router, http.MethodPost, "/api/v1/login", map[string]string{"email": email, "password": password})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/login = %d %s, want 200", rec.Code, rec.Body.String())
	}
	return decodeAuthResponse(t, rec)
}

// registerTestAccount registers a fresh account through the real HTTP flow
// and returns its credentials plus the resulting access token and
// refresh-token cookie.
func registerTestAccount(t *testing.T, deps api.Deps, router http.Handler) (email, password, accessToken string, refreshCookie *http.Cookie) {
	t.Helper()
	name, email, username, password := testUser(t, deps)
	rec := doJSON(t, router, http.MethodPost, "/api/v1/register", map[string]string{
		"name": name, "email": email, "username": username,
		"password": password, "passwordConfirm": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/register = %d %s, want 200", rec.Code, rec.Body.String())
	}
	accessToken, refreshCookie = decodeAuthResponse(t, rec)
	return email, password, accessToken, refreshCookie
}

func authedTokenRequest(method, path, token string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestSettingsShowsProfileAndCurrentSession(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, _, accessToken, refreshCookie := registerTestAccount(t, deps, router)

	req := authedTokenRequest(http.MethodGet, "/api/v1/settings", accessToken)
	req.AddCookie(refreshCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/settings = %d %s, want 200", rec.Code, rec.Body.String())
	}

	settings := decodeJSON[struct {
		Email    string `json:"email"`
		Sessions []struct {
			ID        string `json:"id"`
			IsCurrent bool   `json:"isCurrent"`
		} `json:"sessions"`
	}](t, rec)
	if settings.Email != email {
		t.Errorf("Email = %q, want %q", settings.Email, email)
	}
	foundCurrent := false
	for _, s := range settings.Sessions {
		if s.ID == refreshCookie.Value {
			foundCurrent = true
			if !s.IsCurrent {
				t.Error("the session matching the request's own refresh-token cookie has IsCurrent = false")
			}
		}
	}
	if !foundCurrent {
		t.Fatal("GET /api/v1/settings did not list the session the request authenticated with")
	}
}

func TestUpdateProfile(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	_, _, accessToken, _ := registerTestAccount(t, deps, router)

	req := authedTokenRequest(http.MethodPatch, "/api/v1/settings/profile", accessToken)
	req.Body = jsonBody(t, map[string]string{"name": "New Name", "username": "renameduser"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PATCH /api/v1/settings/profile = %d %s, want 204", rec.Code, rec.Body.String())
	}

	req = authedTokenRequest(http.MethodGet, "/api/v1/settings", accessToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	settings := decodeJSON[struct{ Name string }](t, rec)
	if settings.Name != "New Name" {
		t.Errorf("Name after update = %q, want %q", settings.Name, "New Name")
	}
}

func TestUpdatePasswordRevokesOtherSessions(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, password, accessTokenA, refreshCookieA := registerTestAccount(t, deps, router)

	// A second login creates a second, independent refresh-token session.
	_, refreshCookieB := loginAndGetRefreshCookie(t, deps, router, email, password)

	const newPassword = "a-different-s3cret"
	req := authedTokenRequest(http.MethodPatch, "/api/v1/settings/password", accessTokenA)
	req.AddCookie(refreshCookieA)
	req.Body = jsonBody(t, map[string]string{
		"currentPassword": password, "newPassword": newPassword, "newPasswordConfirm": newPassword,
	})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PATCH /api/v1/settings/password = %d %s, want 204", rec.Code, rec.Body.String())
	}

	// Session A (the one that changed the password) still refreshes.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(refreshCookieA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("POST /api/v1/refresh with the changing session's cookie = %d, want 200 (it must survive its own password change)", rec.Code)
	}

	// Session B was revoked as a side effect.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(refreshCookieB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /api/v1/refresh with the other session's cookie = %d, want 401 (password change should revoke it)", rec.Code)
	}
}

func TestUpdatePasswordRejectsWrongCurrentPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	_, _, accessToken, _ := registerTestAccount(t, deps, router)

	req := authedTokenRequest(http.MethodPatch, "/api/v1/settings/password", accessToken)
	req.Body = jsonBody(t, map[string]string{
		"currentPassword": "totally-wrong", "newPassword": "a-new-password", "newPasswordConfirm": "a-new-password",
	})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("PATCH /api/v1/settings/password (wrong current password) = %d %s, want 401", rec.Code, rec.Body.String())
	}
}

func TestDeleteAccount(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, password, accessToken, refreshCookie := registerTestAccount(t, deps, router)

	req := authedTokenRequest(http.MethodPost, "/api/v1/settings/delete-account", accessToken)
	req.AddCookie(refreshCookie)
	req.Body = jsonBody(t, map[string]string{"currentPassword": password})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /api/v1/settings/delete-account = %d %s, want 204", rec.Code, rec.Body.String())
	}

	// The email is released and the account is really gone -- querying it
	// directly confirms the hard delete, not just that the API 404s.
	var count int
	if err := deps.DB.QueryRow(context.Background(), "SELECT count(*) FROM users WHERE email = $1", email).Scan(&count); err != nil {
		t.Fatalf("query users: %v", err)
	}
	if count != 0 {
		t.Errorf("users row for %q still exists after account deletion", email)
	}
}

func TestRevokeOtherSessions(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, password, accessTokenA, refreshCookieA := registerTestAccount(t, deps, router)
	_, refreshCookieB := loginAndGetRefreshCookie(t, deps, router, email, password)

	req := authedTokenRequest(http.MethodPost, "/api/v1/settings/sessions/revoke-others", accessTokenA)
	req.AddCookie(refreshCookieA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /api/v1/settings/sessions/revoke-others = %d %s, want 204", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(refreshCookieA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("session A (the caller) after revoke-others = %d, want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(refreshCookieB)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("session B after revoke-others = %d, want 401", rec.Code)
	}
}

func TestUpdateTheme(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	_, _, accessToken, _ := registerTestAccount(t, deps, router)

	req := authedTokenRequest(http.MethodPut, "/api/v1/settings/theme", accessToken)
	req.Body = jsonBody(t, map[string]string{"theme": "dark"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT /api/v1/settings/theme (dark) = %d %s, want 204", rec.Code, rec.Body.String())
	}

	req = authedTokenRequest(http.MethodPut, "/api/v1/settings/theme", accessToken)
	req.Body = jsonBody(t, map[string]string{"theme": "not-a-real-theme"})
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT /api/v1/settings/theme (invalid) = %d, want 400", rec.Code)
	}
}
