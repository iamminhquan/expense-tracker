package api

import (
	"net/http"
	"strings"

	"expensetracker/internal/auth"

	"github.com/gin-gonic/gin"
)

// userIDContextKey is the gin.Context key RequireAuth stores the
// authenticated user's ID under; UserID reads it back. A package-private
// type rather than a bare string, so another package's identical-looking
// key can never collide with this one.
type contextKey string

const userIDContextKey contextKey = "userID"

// RequireAuth rejects a request with no valid access token in its
// Authorization header, and for one that has one, stores the user ID
// ParseAccessToken returns so later handlers can read it with UserID.
//
// Unlike the HTML side's auth.RequireAuth, there is no session cookie and
// no database lookup here: the access token is stateless by design (see
// jwt.go), so an expired or tampered token is rejected on its signature
// and exp claim alone. A revoked *refresh* token has no bearing on an
// access token already issued -- it just stops a new one being minted,
// which is the tradeoff a short AccessTokenTTL exists to bound.
func RequireAuth(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed Authorization header"})
			return
		}
		userID, err := auth.ParseAccessToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired access token"})
			return
		}
		c.Set(string(userIDContextKey), userID)
		c.Next()
	}
}

// UserID returns the authenticated user's ID that RequireAuth stored, and
// false if called outside a RequireAuth-protected route.
func UserID(c *gin.Context) (int64, bool) {
	v, ok := c.Get(string(userIDContextKey))
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}
