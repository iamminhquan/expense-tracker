package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewRouter constructs the JSON API's Gin engine: every endpoint under
// /api, plus /healthz at the root for Render's health check and the
// keep-alive cron (see render.yaml), which probe it by that exact path.
func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.CustomRecovery(func(c *gin.Context, _ any) {
		respondError(c, http.StatusInternalServerError, "internal server error")
	}))
	r.Use(corsMiddleware(deps.CORSAllowedOrigins))

	r.HandleMethodNotAllowed = true
	r.NoRoute(func(c *gin.Context) { respondError(c, http.StatusNotFound, "not found") })
	r.NoMethod(func(c *gin.Context) { respondError(c, http.StatusMethodNotAllowed, "method not allowed") })

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	api := r.Group("/api/v1")
	{
		api.POST("/register", registerHandler(deps))
		api.POST("/login", loginHandler(deps))
		api.POST("/refresh", refreshHandler(deps))
		api.POST("/logout", logoutHandler(deps))
		api.POST("/forgot-password", forgotPasswordHandler(deps))
		api.GET("/reset-password", checkResetTokenHandler(deps))
		api.POST("/reset-password", resetPasswordHandler(deps))
		api.POST("/verify-email", verifyEmailHandler(deps))

		authed := api.Group("")
		authed.Use(RequireAuth(deps.JWTSecret))
		{
			authed.GET("/me", meHandler(deps))

			authed.GET("/categories", listCategoriesHandler(deps))
			authed.POST("/categories", createCategoryHandler(deps))
			authed.PATCH("/categories/:id", updateCategoryHandler(deps))
			authed.DELETE("/categories/:id", deleteCategoryHandler(deps))

			authed.GET("/transactions", listTransactionsHandler(deps))
			authed.POST("/transactions", createTransactionHandler(deps))
			authed.PATCH("/transactions/:id", updateTransactionHandler(deps))
			authed.DELETE("/transactions/:id", deleteTransactionHandler(deps))
			authed.GET("/transactions/export", exportTransactionsHandler(deps))
			authed.POST("/transactions/import", importTransactionsHandler(deps))

			authed.GET("/dashboard", dashboardHandler(deps))

			authed.GET("/settings", settingsHandler(deps))
			authed.PATCH("/settings/profile", updateProfileHandler(deps))
			authed.PATCH("/settings/email", updateEmailHandler(deps))
			authed.POST("/settings/resend-verification", resendVerificationHandler(deps))
			authed.PATCH("/settings/password", updatePasswordHandler(deps))
			authed.POST("/settings/delete-account", deleteAccountHandler(deps))
			authed.DELETE("/settings/sessions/:id", revokeSessionHandler(deps))
			authed.POST("/settings/sessions/revoke-others", revokeOtherSessionsHandler(deps))
			authed.PUT("/settings/theme", updateThemeHandler(deps))
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
// An empty origin list (the default until CORS_ALLOWED_ORIGINS is set to
// the real Vercel domain) is a no-op rather than a call to cors.New: that
// library panics on an empty AllowOrigins ("all origins disabled") since
// it assumes a deployment that mounts it always means to allow something.
// No CORS headers at all has the same practical effect here -- a browser
// still refuses any cross-origin request without them -- and doesn't
// force every test and local run to set the env var.
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
