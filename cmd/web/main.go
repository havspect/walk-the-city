package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/havspect/walk-the-city/assets"
	"github.com/havspect/walk-the-city/internal/config"
	"github.com/havspect/walk-the-city/internal/database"
	"github.com/havspect/walk-the-city/internal/nominatim"
	"github.com/havspect/walk-the-city/internal/trip"
)

type application struct {
	logger          *slog.Logger
	config          *config.Config
	html            *htmlRenderer
	staticFS        fs.FS
	tripRepo        trip.TripService
	nominatimClient *nominatim.Client
	tripGenerator   trip.TripGenerator
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("initializing database", "path", cfg.DBPath)
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	logger.Info("parsing html templates")
	renderer, err := newHTMLRenderer(assets.HTMLFiles, "base.tmpl", "partials/*.tmpl")
	if err != nil {
		logger.Error("failed to parse templates", "error", err)
		os.Exit(1)
	}

	app := &application{
		logger:          logger,
		config:          cfg,
		html:            renderer,
		staticFS:        assets.StaticFiles,
		tripRepo:        trip.NewService(db),
		nominatimClient: nominatim.NewClient(),
		tripGenerator:   trip.NewMockGenerator(),
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      app.routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown channel
	shutdownError := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownError <- srv.Shutdown(ctx)
	}()

	logger.Info("starting server", "addr", srv.Addr, "env", cfg.Env)
	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	err = <-shutdownError
	if err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	// Close database connection cleanly to flush WAL
	if sqlDB, err := db.DB(); err == nil {
		if closeErr := sqlDB.Close(); closeErr != nil {
			logger.Warn("failed to close database", "error", closeErr)
		}
	}

	logger.Info("server stopped gracefully")
}
