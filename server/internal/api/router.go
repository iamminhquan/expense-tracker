package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter constructs the JSON API's Gin engine, mounted entirely under
// /api so it can share a process (and, for now, a port) with the Chi
// router in internal/handlers without any path colliding. cmd/server
// wires both into one http.ServeMux.
func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(corsMiddleware(deps.CORSAllowedOrigins))

	api := r.Group("/api")
	{
		api.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

		api.POST("/register", registerHandler(deps))
		api.POST("/login", loginHandler(deps))
		api.POST("/refresh", refreshHandler(deps))
		api.POST("/logout", logoutHandler(deps))

		authed := api.Group("")
		authed.Use(RequireAuth(deps.JWTSecret))
		{
			authed.GET("/me", meHandler(deps))
		}
	}

	return r
}

// corsMiddleware allows credentialed requests (the refresh-token cookie)
// from exactly the configured origins -- never a wildcard, which
// gin-contrib/cors refuses to combine with AllowCredentials anyway, since
// a wildcard + credentials is the CORS misconfiguration that lets any site
// ride a visitor's cookies.
//
// An empty origin list (the default until Phase 3's cutover sets
// CORS_ALLOWED_ORIGINS to the real Vercel domain) is a no-op rather than a
// call to cors.New: that library panics on an empty AllowOrigins ("all
// origins disabled") since it assumes a deployment that mounts it always
// means to allow something. No CORS headers at all has the same practical
// effect here -- a browser still refuses any cross-origin request without
// them -- and doesn't force every test and local run to set the env var.
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
