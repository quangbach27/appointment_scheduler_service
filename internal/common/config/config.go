package config

import (
	"os"
	"strings"
)

type Config struct {
	App *AppConfig
	DB  *DBConfig
}

type AppConfig struct {
	Port               string
	Env                string
	CorsAllowedOrigins []string
}

type DBConfig struct {
	Username string
	Password string
	Host     string
	DBName   string
	Port     string
	URL      string
}

func NewConfig() *Config {
	allowedOrigins := []string{"*"}

	if originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS"); originsEnv != "" {
		origins := strings.Split(originsEnv, ";")
		allowedOrigins = make([]string, 0, len(origins))

		for _, origin := range origins {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	dbUsername := getEnv("DB_USERNAME", "user")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbHost := getEnv("DB_HOST", "localhost")
	dbName := getEnv("DB_NAME", "sumni-finance")
	dbPort := getEnv("DB_PORT", "5432")

	return &Config{
		App: &AppConfig{
			Port:               getEnv("PORT", "4000"),
			Env:                getEnv("APP_ENV", "dev"),
			CorsAllowedOrigins: allowedOrigins,
		},
		DB: &DBConfig{
			Username: dbUsername,
			Password: dbPassword,
			Host:     dbHost,
			DBName:   dbName,
			Port:     dbPort,
		},
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
