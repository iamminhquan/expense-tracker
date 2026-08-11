package config

import "os"

type Config struct {
	DatabaseURL       string
	Port              string
	SessionCookieName string
}

func Load() Config {
	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://expense:expense@localhost:5432/expense_tracker?sslmode=disable"),
		Port:              getEnv("PORT", "8080"),
		SessionCookieName: getEnv("SESSION_COOKIE_NAME", "session_id"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
