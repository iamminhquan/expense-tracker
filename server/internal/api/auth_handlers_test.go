package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"expensetracker/internal/api"
	"expensetracker/internal/database"
	"expensetracker/internal/mailer"
	"expensetracker/internal/sqlcgen"
)

// newTestDeps mirrors handlers_test.newTestDeps, minus the Templates field
// this package has no use for.
func newTestDeps(t *testing.T) api.Deps {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := database.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return api.Deps{
		DB:                pool,
		Queries:           sqlcgen.New(pool),
		Mailer:            mailer.New(mailer.Config{}),
		RefreshCookieName: "session_id",
		SecureCookies:     false,
		BaseURL:           "http://localhost:8080",
		JWTSecret:         []byte("test-secret"),
	}
}

// testUser returns a name/email/username deterministic for t.Name(), so
// parallel tests never collide on a unique constraint, plus a cleanup that
// deletes the row (and, via ON DELETE CASCADE, its sessions) once the test
// ends.
func testUser(t *testing.T, deps api.Deps) (name, email, username, password string) {
	t.Helper()
	h := fnv.New32a()
	h.Write([]byte(t.Name()))
	username = "api" + strconv.FormatUint(uint64(h.Sum32()), 36)
	email = username + "@example.com"
	_, _ = deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	t.Cleanup(func() { _, _ = deps.DB.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email) })
	return "API Test", email, username, "s3cret-pass"
}

func doJSON(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// decodeAccessToken reads the accessToken field any auth endpoint's
// response carries, failing the test if it's missing.
func decodeAccessToken(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	got := decodeData[struct {
		AccessToken string `json:"accessToken"`
	}](t, rec)
	if got.AccessToken == "" {
		t.Fatalf("response %q carries no accessToken", rec.Body.String())
	}
	return got.AccessToken
}

// decodeAuthResponse is decodeAccessToken plus the refresh-token cookie a
// register/login response (but not a refresh response -- refresh
// deliberately doesn't rotate the refresh token, see refreshHandler's doc
// comment) must also set.
func decodeAuthResponse(t *testing.T, rec *httptest.ResponseRecorder) (accessToken string, refreshCookie *http.Cookie) {
	t.Helper()
	accessToken = decodeAccessToken(t, rec)
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" {
			refreshCookie = c
		}
	}
	if refreshCookie == nil {
		t.Fatalf("response set no session_id refresh-token cookie; got cookies %v", rec.Result().Cookies())
	}
	return accessToken, refreshCookie
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	name, email, username, password := testUser(t, deps)

	// Register issues an access token and a refresh-token cookie.
	rec := doJSON(t, router, http.MethodPost, "/api/v1/register", map[string]string{
		"name": name, "email": email, "username": username,
		"password": password, "passwordConfirm": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/register = %d %s, want 200", rec.Code, rec.Body.String())
	}
	accessToken, refreshCookie := decodeAuthResponse(t, rec)

	// That access token authenticates GET /api/v1/me.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/me = %d %s, want 200", rec.Code, rec.Body.String())
	}
	me := decodeData[struct {
		Email string `json:"email"`
	}](t, rec)
	if me.Email != email {
		t.Errorf("GET /api/v1/me email = %q, want %q", me.Email, email)
	}

	// Logging in again (fresh request, no access token yet) works too, and
	// issues its own independent refresh-token cookie.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/login", map[string]string{
		"email": email, "password": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/login = %d %s, want 200", rec.Code, rec.Body.String())
	}
	decodeAuthResponse(t, rec)

	// The refresh-token cookie from registration exchanges for a new
	// access token without resending the password.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(refreshCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/refresh = %d %s, want 200", rec.Code, rec.Body.String())
	}
	decodeAccessToken(t, rec)

	// Logout revokes that refresh token.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	req.AddCookie(refreshCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/logout = %d, want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(refreshCookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /api/v1/refresh after logout = %d, want 401 (refresh token should be revoked)", rec.Code)
	}
	decodeError(t, rec)
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	name, email, username, password := testUser(t, deps)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/register", map[string]string{
		"name": name, "email": email, "username": username,
		"password": password, "passwordConfirm": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/register = %d %s, want 200", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/login", map[string]string{
		"email": email, "password": "wrong-password",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/v1/login (wrong password) = %d %s, want 401", rec.Code, rec.Body.String())
	}
	decodeError(t, rec)
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	name, email, username, password := testUser(t, deps)

	body := map[string]string{
		"name": name, "email": email, "username": username,
		"password": password, "passwordConfirm": password,
	}
	if rec := doJSON(t, router, http.MethodPost, "/api/v1/register", body); rec.Code != http.StatusOK {
		t.Fatalf("first POST /api/v1/register = %d %s, want 200", rec.Code, rec.Body.String())
	}

	body["username"] = username + "2"
	rec := doJSON(t, router, http.MethodPost, "/api/v1/register", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second POST /api/v1/register (duplicate email) = %d %s, want 409", rec.Code, rec.Body.String())
	}
	decodeError(t, rec)
}

func TestMeRequiresAuthentication(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/v1/me with no Authorization header = %d, want 401", rec.Code)
	}
	decodeError(t, rec)
}

func refreshWith(router http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRefreshRotatesTheRefreshToken(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	_, _, _, cookie := registerTestAccount(t, deps, router)

	rec := refreshWith(router, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/refresh = %d %s, want 200", rec.Code, rec.Body.String())
	}
	_, rotated := decodeAuthResponse(t, rec)
	if rotated.Value == cookie.Value {
		t.Fatal("refresh set the same refresh token back, want a new one")
	}
	if !rotated.HttpOnly {
		t.Error("the rotated refresh token is not httpOnly")
	}

	// Its successor works, and rotates in turn.
	if rec := refreshWith(router, rotated); rec.Code != http.StatusOK {
		t.Fatalf("refresh with the new token = %d %s, want 200", rec.Code, rec.Body.String())
	}
}

// A second tab refreshing with the token the first one just replaced is the
// normal race, and must not sign anyone out.
func TestRefreshWithTheJustReplacedTokenIsAnswered(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	_, _, _, cookie := registerTestAccount(t, deps, router)

	_, rotated := decodeAuthResponse(t, refreshWith(router, cookie))
	rec := refreshWith(router, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("racing refresh with the replaced token = %d %s, want 200", rec.Code, rec.Body.String())
	}
	if _, again := decodeAuthResponse(t, rec); again.Value != rotated.Value {
		t.Errorf("racing refresh was handed %q, want the token the first one got (%q)", again.Value, rotated.Value)
	}
}

func TestReusedRefreshTokenEndsTheSession(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	_, _, _, cookie := registerTestAccount(t, deps, router)

	_, rotated := decodeAuthResponse(t, refreshWith(router, cookie))
	if _, err := deps.DB.Exec(context.Background(),
		"UPDATE sessions SET rotated_at = now() - interval '1 hour' WHERE id = $1", rotated.Value); err != nil {
		t.Fatal(err)
	}

	rec := refreshWith(router, cookie)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh token = %d %s, want 401", rec.Code, rec.Body.String())
	}
	decodeError(t, rec)
	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("the refresh-token cookie was not cleared after a reuse")
	}
	if rec := refreshWith(router, rotated); rec.Code != http.StatusUnauthorized {
		t.Errorf("the owner's newer token after a reuse = %d, want 401: the whole session must end", rec.Code)
	}
}

// The client shows its "confirm your email" reminder from this field alone.
func TestUserCarriesEmailVerified(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	email, _, accessToken, _ := registerTestAccount(t, deps, router)

	me := func() bool {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, authedTokenRequest(http.MethodGet, "/api/v1/me", accessToken))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/me = %d %s, want 200", rec.Code, rec.Body.String())
		}
		return decodeData[struct {
			EmailVerified bool `json:"emailVerified"`
		}](t, rec).EmailVerified
	}

	if me() {
		t.Fatal("a freshly registered account reads as verified")
	}
	if _, err := deps.DB.Exec(context.Background(), "UPDATE users SET email_verified = true WHERE email = $1", email); err != nil {
		t.Fatal(err)
	}
	if !me() {
		t.Error("a verified account reads as unverified")
	}
}
