package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

// CookieName is the name of the CSRF token cookie.
const CookieName = "csrf_token"

// HeaderName is the HTTP header name carrying the CSRF token on XHR requests.
const HeaderName = "X-CSRF-Token"

// FormField is the hidden form field name carrying the CSRF token on plain POST submissions.
const FormField = "csrf_token"

// Middleware implements the stateless double-submit-cookie CSRF pattern: no
// server-side token storage is needed because the check only proves the
// request carries back a value the server itself set as a cookie on an
// earlier same-site request. It issues a token cookie on any request that
// doesn't already have one, and rejects state-changing requests whose
// submitted token doesn't match the cookie.
func Middleware(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := ensureCookie(w, r, secure)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			if isMutating(r.Method) {
				submitted := r.Header.Get(HeaderName)
				if submitted == "" {
					// Plain <form method="POST"> submissions (logout) can't
					// set a custom header, so fall back to a hidden field
					// carrying the same token.
					submitted = r.FormValue(FormField)
				}
				if submitted == "" || subtle.ConstantTimeCompare([]byte(submitted), []byte(token)) != 1 {
					http.Error(w, "invalid or missing CSRF token", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ensureCookie returns the request's current CSRF token, generating and
// setting a fresh one on both the response and the in-flight request if
// none exists yet. Patching r (not just w) means the very first page a
// visitor loads already has a valid token to embed via TokenFromRequest,
// instead of only becoming available starting from their second request.
func ensureCookie(w http.ResponseWriter, r *http.Request, secure bool) (string, error) {
	if cookie, err := r.Cookie(CookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	return token, nil
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

// TokenFromRequest reads the current request's CSRF token for embedding
// into a rendered page (meta tag, hidden form input). Middleware guarantees
// a token is present on the request by the time handlers run.
func TokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
