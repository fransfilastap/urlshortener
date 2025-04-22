package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the global logger with the specified configuration
func InitLogger(level, format string) {
	// Set up time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Set up log level
	logLevel := getLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// Set up log output format
	var output io.Writer = os.Stdout
	if strings.ToLower(format) == "console" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// Set global logger
	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

	// Log initialization
	log.Info().
		Str("level", level).
		Str("format", format).
		Msg("Logger initialized")
}

// getLogLevel converts a string log level to zerolog.Level
func getLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// EchoLogger returns a middleware function that logs HTTP requests
func EchoLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogMethod: true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info().
				Str("uri", v.URI).
				Int("status", v.Status).
				Str("method", v.Method).
				Dur("latency", v.Latency).
				Str("remote_ip", c.RealIP()).
				Str("user_agent", c.Request().UserAgent()).
				Msg("request")
			return nil
		},
	})
}

// Get returns the global logger
func Get() *zerolog.Logger {
	return &log.Logger
}