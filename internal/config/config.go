package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	ServerPort string
	BaseURL    string
	APIKey     string

	PostgresURL string

	ValkeyCacheAddr     string
	ValkeyCachePassword string
	ValkeyCacheDB       int
	ValkeyCacheTTL      time.Duration

	LogLevel  string
	LogFormat string

	SessionSecret string
	SessionMaxAge int
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load .env file")
	}

	return &Config{
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		BaseURL:             getEnv("BASE_URL", "http://localhost:8080"),
		APIKey:              getEnv("API_KEY", "your-api-key-here"),
		PostgresURL:         getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable"),
		ValkeyCacheAddr:     getEnv("VALKEY_ADDR", "localhost:6379"),
		ValkeyCachePassword: getEnv("VALKEY_PASSWORD", ""),
		ValkeyCacheDB:       getEnvAsInt("VALKEY_DB", 0),
		ValkeyCacheTTL:      getEnvAsDuration("VALKEY_TTL", 24*time.Hour),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		SessionSecret:        getEnv("SESSION_SECRET", "change-me-in-production"),
		SessionMaxAge:        getEnvAsInt("SESSION_MAX_AGE", 86400),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}