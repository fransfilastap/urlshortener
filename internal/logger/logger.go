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

func InitLogger(level, format string) {
	zerolog.TimeFieldFormat = time.RFC3339

	logLevel := getLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	var output io.Writer = os.Stdout
	if strings.ToLower(format) == "console" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

	log.Info().
		Str("level", level).
		Str("format", format).
		Msg("Logger initialized")
}

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

func Get() *zerolog.Logger {
	return &log.Logger
}