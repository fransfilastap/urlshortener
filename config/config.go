package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	// Server settings
	ServerPort string
	BaseURL    string
	APIKey     string

	// Database settings
	PostgresURL string

	// Cache settings
	ValkeyCacheAddr     string
	ValkeyCachePassword string
	ValkeyCacheDB       int
	ValkeyCacheTTL      time.Duration

	// Logging settings
	LogLevel  string
	LogFormat string
}

// NewConfig creates a new configuration with values from environment variables
func NewConfig() *Config {
	return &Config{
		// Server settings
		ServerPort: getEnv("SERVER_PORT", "8080"),
		BaseURL:    getEnv("BASE_URL", "http://localhost:8080"),
		APIKey:     getEnv("API_KEY", "your-api-key-here"),

		// Database settings
		PostgresURL: getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable"),

		// Cache settings
		ValkeyCacheAddr:     getEnv("VALKEY_ADDR", "localhost:6379"),
		ValkeyCachePassword: getEnv("VALKEY_PASSWORD", ""),
		ValkeyCacheDB:       getEnvAsInt("VALKEY_DB", 0),
		ValkeyCacheTTL:      getEnvAsDuration("VALKEY_TTL", 24*time.Hour),

		// Logging settings
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsDuration gets an environment variable as a duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
