package main

import (
	"fmt"
	"net/http"
	"os"
	"url-shortener/internal/config"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage/sqlite"

	mwLogger "url-shortener/internal/http-server/middleware/logger"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/lib/handlers/slogpretty"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
)

func main() {
	os.Setenv("CONFIG_PATH", "./config/local.yaml")
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	log = log.With(slog.String("env", cfg.Env))

	log.Info("Application started")
	log.Error("Debug configuration:")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("Failed to create storage: ", sl.Err(err))
		os.Exit(1)
	}
	log.Info("Storage created successfully")

	router := chi.NewRouter()
	// Add routes to router here...
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
		}))

		r.Post("/", save.New(log, storage))
		// TODO: add DELETE /url/{id}
	})
	router.Get("/{alias}", redirect.New(log, storage))

	log.Info("HTTP server started", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("HTTP server failed to start: ", sl.Err(err))
		os.Exit(1)
	}

}

const (
	envLocal = "local"
	envProd  = "prod"
	envdev   = "dev"
)

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envdev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log

}

func setupPrettySlog() *slog.Logger {
	fmt.Printf("Setting up pretty slog logger...\n")
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
