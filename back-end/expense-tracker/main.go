package main

import (
	"log"

	"github.com/iamminhquan/expense-tracker/internal/config"
	"github.com/iamminhquan/expense-tracker/internal/database"
	"github.com/iamminhquan/expense-tracker/internal/routes"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get database instance: %v", err)
	}
	defer sqlDB.Close()

	router := routes.Setup(db, cfg)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
