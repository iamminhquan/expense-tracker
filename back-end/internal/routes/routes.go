package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iamminhquan/expense-tracker/internal/handlers"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB) *gin.Engine {
	router := gin.Default()
	healthHandler := handlers.NewHealthHandler(db)

	router.GET("/health", healthHandler.Check)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
		})

		v1.GET("/hello/:name", func(c *gin.Context) {
			name := c.DefaultQuery("name", "World")
			c.JSON(http.StatusOK, gin.H{"message": "Hello, " + name + "!"})
		})
	}

	return router
}
