package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/iamminhquan/expense-tracker/internal/config"
	"github.com/iamminhquan/expense-tracker/internal/handlers"
	"github.com/iamminhquan/expense-tracker/internal/middleware"
	"github.com/iamminhquan/expense-tracker/internal/repositories"
	"github.com/iamminhquan/expense-tracker/internal/services"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg config.Config) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CORS(cfg.CORSOrigin))

	healthHandler := handlers.NewHealthHandler(db)
	userRepo := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepo, cfg.AuthSecret)
	userHandler := handlers.NewUserHandler(userService)

	router.GET("/health", healthHandler.Check)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/users", userHandler.Register)
		v1.POST("/auth/register", userHandler.Register)
		v1.POST("/auth/login", userHandler.Login)
		v1.GET("/auth/me", middleware.AuthRequired(cfg), userHandler.Me)
	}

	return router
}
