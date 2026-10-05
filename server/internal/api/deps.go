// Package api is the Gin-routed JSON API ("/api/v1/*") that is the whole
// backend -- it replaced the Chi-routed, html/template-rendered
// internal/handlers package, which no longer exists (see
// .claude/context/server.md for the current architecture and git history
// for what the old app looked like).
package api

import (
	"expensetracker/internal/mailer"
	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps holds the shared dependencies for the JSON API handlers -- the
// /api/v1/* counterpart to handlers.Deps. It carries no Templates field (this
// package never renders HTML) and adds the JWT/CORS config the HTML side
// has no use for.
type Deps struct {
	DB      *pgxpool.Pool
	Queries *sqlcgen.Queries
	Mailer  *mailer.Mailer
	// RefreshCookieName names the httpOnly cookie carrying the refresh
	// token. It is session.go's existing session token, reused as a
	// refresh token rather than a second, parallel revocable-token
	// mechanism -- see jwt.go's doc comment for why.
	RefreshCookieName string
	// SecureCookies gates the Secure attribute on the refresh-token
	// cookie; see internal/config.Config.SecureCookies.
	SecureCookies bool
	// BaseURL builds the absolute link in a verification email; see
	// internal/config.Config.BaseURL.
	BaseURL string
	// JWTSecret signs and verifies access tokens; see
	// internal/config.Config.JWTSecret. Required -- config.Load() refuses
	// to start without it.
	JWTSecret []byte
	// CORSAllowedOrigins lists the origins the router's CORS middleware
	// accepts credentialed cross-origin requests from; see
	// internal/config.Config.CORSAllowedOrigins.
	CORSAllowedOrigins []string
}
