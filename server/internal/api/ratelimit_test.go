package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expensetracker/internal/api"
)

// loginFrom sends a login with an unparseable body, so the handler answers 400
// without touching the database; only the rate limiter can turn it into a 429.
func loginFrom(router http.Handler, remoteAddr, forwardedFor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestLoginIsRateLimitedPerIP(t *testing.T) {
	router := api.NewRouter(api.Deps{})

	for i := range 10 {
		if rec := loginFrom(router, "203.0.113.5:4000", ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("request %d = %d, want 400 (inside the burst)", i+1, rec.Code)
		}
	}
	rec := loginFrom(router, "203.0.113.5:4000", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request 11 = %d, want 429; body %q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Retry-After"); got != "6" {
		t.Errorf("Retry-After = %q, want %q", got, "6")
	}
	decodeError(t, rec)

	if rec := loginFrom(router, "203.0.113.6:4000", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("another IP = %d, want 400: one client's budget must not drain another's", rec.Code)
	}
}

// A client must not be able to pick the address it is limited under by
// sending its own X-Forwarded-For.
func TestForwardedForIsIgnoredFromAnUntrustedPeer(t *testing.T) {
	router := api.NewRouter(api.Deps{})

	for i := range 10 {
		loginFrom(router, "203.0.113.5:4000", "198.51.100."+string(rune('1'+i)))
	}
	if rec := loginFrom(router, "203.0.113.5:4000", "198.51.100.99"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rotating X-Forwarded-For dodged the limit: got %d, want 429", rec.Code)
	}
}

// Behind a trusted proxy the client is the address the proxy reports, not the
// proxy's own, or every user would share one budget.
func TestForwardedForIsUsedFromATrustedProxy(t *testing.T) {
	router := api.NewRouter(api.Deps{TrustedProxies: []string{"10.0.0.0/8"}})

	for range 10 {
		loginFrom(router, "10.1.2.3:4000", "198.51.100.1")
	}
	if rec := loginFrom(router, "10.1.2.3:4000", "198.51.100.1"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("same forwarded client = %d, want 429", rec.Code)
	}
	if rec := loginFrom(router, "10.1.2.3:4000", "198.51.100.2"); rec.Code != http.StatusBadRequest {
		t.Errorf("different forwarded client = %d, want 400", rec.Code)
	}
}

func postFrom(router http.Handler, path, remoteAddr string) int {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{"))
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code
}

// Register and forgot-password draw on one budget (both create an account or
// send an email), separate from the one login guesses spend.
func TestSignupBudgetIsSharedAndSeparateFromLogin(t *testing.T) {
	router := api.NewRouter(api.Deps{})
	const ip = "203.0.113.5:4000"

	for range 10 {
		loginFrom(router, ip, "")
	}
	for i := range 5 {
		path := "/api/v1/forgot-password"
		if i == 0 {
			path = "/api/v1/register"
		}
		if got := postFrom(router, path, ip); got != http.StatusBadRequest {
			t.Fatalf("%s #%d after a spent login budget = %d, want 400", path, i+1, got)
		}
	}
	if got := postFrom(router, "/api/v1/register", ip); got != http.StatusTooManyRequests {
		t.Fatalf("register after 5 signup-budget requests = %d, want 429", got)
	}
	if got := postFrom(router, "/api/v1/forgot-password", ip); got != http.StatusTooManyRequests {
		t.Fatalf("forgot-password after 5 signup-budget requests = %d, want 429", got)
	}
}
