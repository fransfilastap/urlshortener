package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fransfilastap/urlshortener"
	"github.com/fransfilastap/urlshortener/internal/config"
	"github.com/fransfilastap/urlshortener/internal/handler"
	"github.com/fransfilastap/urlshortener/internal/logger"
	"github.com/fransfilastap/urlshortener/internal/repository"
	"github.com/fransfilastap/urlshortener/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.NewConfig()

	logger.InitLogger(cfg.LogLevel, cfg.LogFormat)

	db, err := repository.NewPostgresRepository(cfg.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := repository.RunMigrations(cfg.PostgresURL, urlshortener.MigrationsFS); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	cache := repository.NewCacheRepository(
		cfg.ValkeyCacheAddr,
		cfg.ValkeyCachePassword,
		cfg.ValkeyCacheDB,
		cfg.ValkeyCacheTTL,
	)
	defer cache.Close()

	urlService := service.NewURLService(db, cache)
	sessionStore := handler.GetSessionStore(cfg.SessionSecret, cfg.SessionMaxAge)
	authHandler := handler.NewAuthHandler(sessionStore, cfg.APIKey)

	e := echo.New()

	e.Use(logger.EchoLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Auth endpoints
	e.POST("/auth/login", authHandler.Login)
	e.POST("/auth/logout", authHandler.Logout)
	e.GET("/auth/me", authHandler.Me)

	// Public redirect endpoint
	urlHandler := handler.NewURLHandler(urlService, cfg.BaseURL, cfg.APIKey)
	e.GET("/:code", urlHandler.RedirectURL)

	// API endpoints — dual auth (session cookie OR API key header)
	apiGroup := e.Group("")
	apiGroup.Use(handler.SessionOrAPIKeyMiddleware(sessionStore, cfg.APIKey))
	apiGroup.POST("/api/shorten", urlHandler.ShortenURL)
	apiGroup.GET("/api/urls/:code", urlHandler.GetURLInfo)
	apiGroup.PUT("/api/urls/:code", urlHandler.UpdateURL)
	apiGroup.DELETE("/api/urls/:code", urlHandler.DeleteURL)
	apiGroup.GET("/api/urls/:code/analytics", urlHandler.GetURLAnalytics)
	apiGroup.GET("/api/urls/creator/:creator_reference", urlHandler.GetURLsByCreator)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	go func() {
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server shutdown failed")
	}

	log.Info().Msg("Server gracefully stopped")
}