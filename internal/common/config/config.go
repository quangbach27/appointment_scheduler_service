// Package config
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	App         *AppConfig
	Appointment *AppointmentConfig
	DB          *DBConfig
}

type AppConfig struct {
	Port               string
	Env                string
	CorsAllowedOrigins []string
}

type AppointmentConfig struct {
	ReservationTTL   time.Duration
	MinStartLeadTime time.Duration
	MaxStartLeadTime time.Duration
}

type DBConfig struct {
	Username string
	Password string
	Host     string
	DBName   string
	Port     string
	URL      string
}

var (
	configOnce sync.Once
	config     *Config
)

func NewConfig() *Config {
	configOnce.Do(func() {
		config = &Config{
			App:         newAppConfig(),
			Appointment: newAppointmentConfig(),
			DB:          newDBConfig(),
		}
	})

	return config
}

func newAppConfig() *AppConfig {
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

	return &AppConfig{
		Port:               getEnv("PORT", "4000"),
		Env:                getEnv("APP_ENV", "dev"),
		CorsAllowedOrigins: allowedOrigins,
	}
}

func newAppointmentConfig() *AppointmentConfig {
	reservationTTLMinutes := getIntEnv("APPOINTMENT_RESERVATION_TTL_MINUTES", 5)
	minStartLeadTimeHours := getIntEnv("APPOINTMENT_MIN_START_LEAD_TIME_HOURS", 1)
	maxStartLeadTimeDays := getIntEnv("APPOINTMENT_MAX_START_LEAD_TIME_DAYS", 30)

	return &AppointmentConfig{
		ReservationTTL:   time.Duration(reservationTTLMinutes) * time.Minute,
		MinStartLeadTime: time.Duration(minStartLeadTimeHours) * time.Hour,
		MaxStartLeadTime: time.Duration(maxStartLeadTimeDays) * 24 * time.Hour,
	}
}

func newDBConfig() *DBConfig {
	dbUsername := getEnv("DB_USERNAME", "user")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbHost := getEnv("DB_HOST", "localhost")
	dbName := getEnv("DB_NAME", "sumni-finance")
	dbPort := getEnv("DB_PORT", "5432")
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUsername, dbPassword, dbHost, dbPort, dbName)

	return &DBConfig{
		Username: dbUsername,
		Password: dbPassword,
		Host:     dbHost,
		DBName:   dbName,
		Port:     dbPort,
		URL:      dbURL,
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		slog.Warn(fmt.Sprintf("warning: invalid value for %s=%q, using fallback %d: %v", key, value, fallback, err))

		return fallback
	}
	return n
}
