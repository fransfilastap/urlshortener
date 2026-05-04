package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrateCommand()
		return
	}

	runServer()
}

func runMigrateCommand() {
	cfg := config.NewConfig()

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: urlshortener migrate <up|version|force> [version]")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "up":
		if err := repository.RunMigrations(cfg.PostgresURL, urlshortener.MigrationsFS); err != nil {
			fmt.Fprintf(os.Stderr, "Error running migrations: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully")
	case "version":
		version, dirty, err := repository.GetSchemaVersion(cfg.PostgresURL, urlshortener.MigrationsFS)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting schema version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Schema version: %d (dirty: %v)\n", version, dirty)
	case "force":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: urlshortener migrate force <version>")
			os.Exit(1)
		}
		version, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid version number: %v\n", err)
			os.Exit(1)
		}
		if err := repository.ForceVersion(cfg.PostgresURL, urlshortener.MigrationsFS, version); err != nil {
			fmt.Fprintf(os.Stderr, "Error forcing version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Forced schema version to %d\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown migrate command: %s\n", os.Args[2])
		fmt.Fprintln(os.Stderr, "Usage: urlshortener migrate <up|version|force> [version]")
		os.Exit(1)
	}
}

func runServer() {
	cfg := config.NewConfig()

	logger.InitLogger(cfg.LogLevel, cfg.LogFormat)

	if cfg.AutoMigrate {
		if err := repository.RunMigrations(cfg.PostgresURL, urlshortener.MigrationsFS); err != nil {
			log.Fatal().Err(err).Msg("Failed to run database migrations")
		}
	} else {
		log.Info().Msg("Migrations skipped (AUTO_MIGRATE=false)")
	}

	if cfg.ExpectedSchemaVersion > 0 {
		if err := repository.ValidateSchemaVersion(cfg.PostgresURL, urlshortener.MigrationsFS, cfg.ExpectedSchemaVersion); err != nil {
			log.Fatal().Err(err).Msg("Schema version check failed")
		}
		log.Info().Int("version", cfg.ExpectedSchemaVersion).Msg("Schema version validated")
	}

	db, err := repository.NewPostgresRepository(cfg.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

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

	// Public landing page and redirect endpoints
	urlHandler := handler.NewURLHandler(urlService, cfg.BaseURL, cfg.APIKey)
	e.GET("/", urlHandler.Index)
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

	// Serve SPA (admin dashboard)
	if handler.IsDistDirPresent() {
		handler.RegisterSPA(e, urlshortener.DistFS)
	}

	// Serve static assets (logo, etc)
	staticFS, _ := fs.Sub(urlshortener.StaticFS, "static")
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static", http.FileServer(http.FS(staticFS)))))

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