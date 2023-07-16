package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"

	"url-shortener/internal/config"
	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/lib/logger/handlers/slogpretty"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage/sqlite"
)

func main() {
	cfg := config.MustLoad()

	log := newSlogLogger(cfg.Log.Slog)

	log.Info(
		"starting url-shortener",
		slog.String("env", cfg.Env),
		slog.String("version", "123"),
	)
	log.Debug("debug messages are enabled")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	_ = storage

	router := chi.NewRouter()

	router.Use(middleware.RequestID)

	// TODO: Log requests with mismatched route using slog instead of chi logger
	router.Use(middleware.Logger)

	router.Use(mwLogger.New(log))

	// TODO: use slog to log panic and github.com/maruel/panicparse to handle it
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

	log.Info("starting server", slog.String("address", cfg.Address))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server: " + err.Error())
	}

	log.Error("server stopped")
}

func newSlogLogger(c config.Slog) *slog.Logger {
	o := &slog.HandlerOptions{Level: c.Level, AddSource: c.AddSource}
	w := os.Stdout
	var h slog.Handler

	switch c.Format {
	case "pretty":
		h = slogpretty.NewHandler().
			WithAddSource(c.AddSource).
			WithLevel(c.Level).
			WithLevelEmoji(c.Pretty.Emoji).
			WithFieldsFormat(c.Pretty.FieldsFormat)
	case "json":
		h = slog.NewJSONHandler(w, o)
	case "text":
		h = slog.NewTextHandler(w, o)
	}
	return slog.New(h)
}
