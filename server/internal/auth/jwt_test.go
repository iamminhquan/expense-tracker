package auth_test

import (
	"testing"
	"time"

	"expensetracker/internal/auth"

	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndParseAccessToken(t *testing.T) {
	secret := []byte("test-secret")

	token, expiresAt, err := auth.IssueAccessToken(42, secret)
	if err != nil {
		t.Fatalf("IssueAccessToken() = _, _, %v, want no error", err)
	}
	if token == "" {
		t.Fatal("IssueAccessToken() returned an empty token")
	}
	wantExpiry := time.Now().Add(auth.AccessTokenTTL)
	if expiresAt.Before(wantExpiry.Add(-time.Second)) || expiresAt.After(wantExpiry.Add(time.Second)) {
		t.Errorf("expiresAt = %v, want close to %v", expiresAt, wantExpiry)
	}

	userID, err := auth.ParseAccessToken(token, secret)
	if err != nil {
		t.Fatalf("ParseAccessToken() = _, %v, want no error", err)
	}
	if userID != 42 {
		t.Errorf("ParseAccessToken() = %d, want 42", userID)
	}
}

func TestParseAccessTokenRejectsWrongSecret(t *testing.T) {
	token, _, err := auth.IssueAccessToken(1, []byte("secret-a"))
	if err != nil {
		t.Fatalf("IssueAccessToken() = _, _, %v, want no error", err)
	}

	if _, err := auth.ParseAccessToken(token, []byte("secret-b")); err == nil {
		t.Error("ParseAccessToken() succeeded with the wrong secret, want an error")
	}
}

func TestParseAccessTokenRejectsExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	claims := jwt.MapClaims{
		"uid": float64(1),
		"exp": jwt.NewNumericDate(time.Now().Add(-time.Minute)),
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	if _, err := auth.ParseAccessToken(expired, secret); err == nil {
		t.Error("ParseAccessToken() succeeded with an expired token, want an error")
	}
}

func TestParseAccessTokenRejectsUnsignedToken(t *testing.T) {
	secret := []byte("test-secret")
	claims := jwt.MapClaims{"uid": float64(1)}
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none-alg token: %v", err)
	}

	if _, err := auth.ParseAccessToken(none, secret); err == nil {
		t.Error("ParseAccessToken() succeeded with an alg=none token, want an error")
	}
}

func TestParseAccessTokenRejectsGarbage(t *testing.T) {
	if _, err := auth.ParseAccessToken("not-a-jwt", []byte("test-secret")); err == nil {
		t.Error("ParseAccessToken() succeeded on a garbage string, want an error")
	}
}
