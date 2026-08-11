package main

import (
	"log"
	"net/http"

	"expensetracker/internal/config"
	"expensetracker/internal/handlers"
)

func main() {
	cfg := config.Load()

	router := handlers.NewRouter(handlers.Deps{})

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
