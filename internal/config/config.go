package config

import (
	"os"
	"strconv"
	"strings"
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

	SessionSecret         string
	SessionMaxAge         int
	AutoMigrate           bool
	ExpectedSchemaVersion int

	AllowedOrigins []string
	SecureCookies  bool
	RateLimit      int
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load .env file")
	}

	apiKey := getEnvRequired("API_KEY")
	sessionSecret := getEnvRequired("SESSION_SECRET")

	secureCookies := getEnvAsBool("SECURE_COOKIES", true)

	rateLimit := getEnvAsInt("RATE_LIMIT", 100)

	allowedOrigins := getEnvAsSlice("ALLOWED_ORIGINS", []string{getEnv("BASE_URL", "http://localhost:8080")})

	return &Config{
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		BaseURL:             getEnv("BASE_URL", "http://localhost:8080"),
		APIKey:              apiKey,
		PostgresURL:         getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable"),
		ValkeyCacheAddr:     getEnv("VALKEY_ADDR", "localhost:6379"),
		ValkeyCachePassword: getEnv("VALKEY_PASSWORD", ""),
		ValkeyCacheDB:       getEnvAsInt("VALKEY_DB", 0),
		ValkeyCacheTTL:      getEnvAsDuration("VALKEY_TTL", 24*time.Hour),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		SessionSecret:       sessionSecret,
		SessionMaxAge:       getEnvAsInt("SESSION_MAX_AGE", 86400),
		AutoMigrate:         getEnvAsBool("AUTO_MIGRATE", true),
		ExpectedSchemaVersion: getEnvAsInt("EXPECTED_SCHEMA_VERSION", 0),
		AllowedOrigins:      allowedOrigins,
		SecureCookies:       secureCookies,
		RateLimit:           rateLimit,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatal().Str("key", key).Msg("Required environment variable is not set")
	}
	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	var result []string
	for _, v := range strings.Split(value, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
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

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}