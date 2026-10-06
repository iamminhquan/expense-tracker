package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

	settings := decodeData[struct {
		Email    string `json:"email"`
		Sessions []struct {
			ID        string `json:"id"`
			IsCurrent bool   `json:"isCurrent"`
		} `json:"sessions"`
	}](t, rec)
	if settings.Email != email {
		t.Errorf("Email = %q, want %q", settings.Email, email)
	}
	currentCount := 0
	for _, s := range settings.Sessions {
		if s.ID == refreshCookie.Value {
			t.Fatalf("the sessions list exposes the refresh token as a session id: %q", s.ID)
		}
		if s.IsCurrent {
			currentCount++
		}
	}
	if currentCount != 1 {
		t.Errorf("%d sessions marked current, want exactly the one matching the request's refresh-token cookie", currentCount)
	}
	if strings.Contains(rec.Body.String(), refreshCookie.Value) {
		t.Errorf("GET /api/v1/settings body contains the refresh token: %s", rec.Body.String())
	}
}

type listedSession struct {
	ID        string `json:"id"`
	IsCurrent bool   `json:"isCurrent"`
}

func listSessions(t *testing.T, router http.Handler, accessToken string, refreshCookie *http.Cookie) []listedSession {
	t.Helper()
	req := authedTokenRequest(http.MethodGet, "/api/v1/settings", accessToken)
	req.AddCookie(refreshCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/settings = %d %s, want 200", rec.Code, rec.Body.String())
	}
	return decodeData[struct {
		Sessions []listedSession `json:"sessions"`
	}](t, rec).Sessions
}

func refreshStatus(router http.Handler, cookie *http.Cookie) int {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code
}

// The public id from the sessions list is what a client sends to revoke a
// session, and it can only ever reach the caller's own.
func TestRevokeSessionByPublicID(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, password, accessTokenA, refreshCookieA := registerTestAccount(t, deps, router)
	_, refreshCookieB := loginAndGetRefreshCookie(t, deps, router, email, password)

	var otherID string
	for _, s := range listSessions(t, router, accessTokenA, refreshCookieA) {
		if !s.IsCurrent {
			otherID = s.ID
		}
	}
	if otherID == "" {
		t.Fatal("no second session in the list")
	}

	// Another account's session id revokes nothing of this account's.
	otherEmail := "other-" + email
	_, _ = deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", otherEmail)
	t.Cleanup(func() { _, _ = deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", otherEmail) })
	rec := doJSON(t, router, http.MethodPost, "/api/v1/register", map[string]string{
		"name": "Other", "email": otherEmail, "username": "o" + strings.TrimSuffix(strings.TrimPrefix(email, "api"), "@example.com"),
		"password": password, "passwordConfirm": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("register second account = %d %s", rec.Code, rec.Body.String())
	}
	accessTokenX, refreshCookieX := decodeAuthResponse(t, rec)
	req := authedTokenRequest(http.MethodDelete, "/api/v1/settings/sessions/"+otherID, accessTokenX)
	req.AddCookie(refreshCookieX)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if got := refreshStatus(router, refreshCookieB); got != http.StatusOK {
		t.Fatalf("session B after another account tried to revoke it = %d, want 200", got)
	}

	req = authedTokenRequest(http.MethodDelete, "/api/v1/settings/sessions/"+otherID, accessTokenA)
	req.AddCookie(refreshCookieA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/v1/settings/sessions/{id} = %d %s, want 200", rec.Code, rec.Body.String())
	}
	if got := refreshStatus(router, refreshCookieB); got != http.StatusUnauthorized {
		t.Errorf("revoked session B refresh = %d, want 401", got)
	}
	if got := refreshStatus(router, refreshCookieA); got != http.StatusOK {
		t.Errorf("session A refresh = %d, want 200", got)
	}

	// The refresh token itself is not an id, so it can't be used as one.
	req = authedTokenRequest(http.MethodDelete, "/api/v1/settings/sessions/"+refreshCookieA.Value, accessTokenA)
	req.AddCookie(refreshCookieA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("DELETE with a refresh token as the id = %d, want 400", rec.Code)
	}
	if got := refreshStatus(router, refreshCookieA); got != http.StatusOK {
		t.Errorf("session A after that attempt = %d, want 200", got)
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
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/v1/settings/profile = %d %s, want 200", rec.Code, rec.Body.String())
	}

	req = authedTokenRequest(http.MethodGet, "/api/v1/settings", accessToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	settings := decodeData[struct{ Name string }](t, rec)
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
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/v1/settings/password = %d %s, want 200", rec.Code, rec.Body.String())
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
	decodeError(t, rec)
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
	decodeError(t, rec)
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
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/settings/delete-account = %d %s, want 200", rec.Code, rec.Body.String())
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
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/settings/sessions/revoke-others = %d %s, want 200", rec.Code, rec.Body.String())
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
	decodeError(t, rec)
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
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/v1/settings/theme (dark) = %d %s, want 200", rec.Code, rec.Body.String())
	}

	req = authedTokenRequest(http.MethodPut, "/api/v1/settings/theme", accessToken)
	req.Body = jsonBody(t, map[string]string{"theme": "not-a-real-theme"})
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT /api/v1/settings/theme (invalid) = %d, want 400", rec.Code)
	}
	decodeError(t, rec)
}
