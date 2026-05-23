package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv     string
	AppPort    string
	DBHost     string
	DBPort     int
	DBUser     string
	DBPass     string
	DBName     string
	DBSSLMode  string
	AuthSecret string
	CORSOrigin string
}

func Load() Config {
	return Config{
		AppEnv:     getEnv("APP_ENV", "development"),
		AppPort:    getEnv("APP_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPass:     getEnv("DB_PASSWORD", "quan"),
		DBName:     getEnv("DB_NAME", "expense_tracker"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		AuthSecret: getEnv("AUTH_SECRET", "development-auth-secret"),
		CORSOrigin: getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:3000"),
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Bangkok",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPass,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
