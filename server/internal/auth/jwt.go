package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessTokenTTL is how long an issued access token stays valid. Kept short
// because the access token is stateless -- unlike a session row, it is
// never checked against the database, so it cannot be revoked before it
// expires on its own. The refresh token (session.go's existing token,
// reused as-is: see its doc comment) is what a logout or "log out
// everywhere else" actually revokes; a short-lived access token just
// bounds how long a compromised one stays useful.
const AccessTokenTTL = 15 * time.Minute

// ErrInvalidAccessToken is returned when a token fails signature
// verification, has expired, or is not an access token this package
// issued.
var ErrInvalidAccessToken = errors.New("invalid access token")

// accessClaims is the JWT payload for an access token. Authorization is
// single-tier (see backend.md's Authorization Model) -- every authenticated
// user has the same capabilities scoped to their own data -- so naming the
// user is the only claim this needs.
type accessClaims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// IssueAccessToken mints a short-lived JWT identifying userID, signed with
// secret (config.Config.JWTSecret).
func IssueAccessToken(userID int64, secret []byte) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(AccessTokenTTL)
	claims := accessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// ParseAccessToken validates tokenString's signature and expiry against
// secret and returns the user ID it names.
func ParseAccessToken(tokenString string, secret []byte) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(t *jwt.Token) (any, error) {
		// Reject anything but HS256 before ever touching secret -- jwt-go's
		// documented defence against an attacker swapping in "alg": "none"
		// or an asymmetric algorithm the verifier would otherwise trust.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidAccessToken
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return 0, ErrInvalidAccessToken
	}
	claims, ok := token.Claims.(*accessClaims)
	if !ok || claims.UserID == 0 {
		return 0, ErrInvalidAccessToken
	}
	return claims.UserID, nil
}
