package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fransfilastap/urlshortener/config"
	"github.com/fransfilastap/urlshortener/handlers"
	"github.com/fransfilastap/urlshortener/logger"
	"github.com/fransfilastap/urlshortener/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Initialize logger
	logger.InitLogger(cfg.LogLevel, cfg.LogFormat)

	// Initialize database
	db, err := store.NewPostgresRepository(cfg.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize database schema
	if err := db.InitSchema(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database schema")
	}

	// Initialize cache
	cache := store.NewCacheRepository(
		cfg.ValkeyCacheAddr,
		cfg.ValkeyCachePassword,
		cfg.ValkeyCacheDB,
		cfg.ValkeyCacheTTL,
	)
	defer cache.Close()

	// Initialize URL service
	urlService := store.NewURLService(db, cache)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(logger.EchoLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Serve static files
	e.Static("/static", "static")

	// Initialize handlers
	urlHandler := handlers.NewURLHandler(urlService, cfg.BaseURL, cfg.APIKey)
	urlHandler.Register(e)

	// Add health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Start server
	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server shutdown failed")
	}

	log.Info().Msg("Server gracefully stopped")
}
