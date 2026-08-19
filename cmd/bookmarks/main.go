// Command bookmarks is the entrypoint for the self-hosted bookmark manager.
package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/Cyclenerd/bookmarks-manager/internal/config"
	"github.com/Cyclenerd/bookmarks-manager/internal/database"
	"github.com/Cyclenerd/bookmarks-manager/internal/handler"
	"github.com/Cyclenerd/bookmarks-manager/internal/middleware"
	"github.com/Cyclenerd/bookmarks-manager/internal/repository"
	"github.com/Cyclenerd/bookmarks-manager/internal/service"
	"github.com/Cyclenerd/bookmarks-manager/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	start := time.Now()
	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	// Fully initialise the application before serving any requests. Only once
	// everything is ready does the server start listening, so clients never see
	// a half-initialised state.
	handler, db, err := initialize(cfg, logger)
	if err != nil {
		return err
	}
	logger.Info("ready", "startup", time.Since(start).String())

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	// Wait for a shutdown signal or an unexpected server error.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serveErr:
		db.Close()
		return err
	case <-stop:
	}

	// Cloud Run sends SIGTERM and allows a short grace period before SIGKILL.
	// Drain in-flight requests first, then close the database. Closing SQLite
	// finalises the rollback journal and closes the underlying file, which is
	// what makes gcsfuse upload the final state to Cloud Storage — critical
	// when the instance can be killed at any time.
	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(ctx)
	if err := db.Close(); err != nil {
		logger.Error("closing database", "err", err)
		if shutdownErr == nil {
			shutdownErr = err
		}
	}
	return shutdownErr
}

// initialize wires up the database, repositories, services, templates and
// routes, returning the fully-configured HTTP handler and the database handle
// (so the caller can close it on shutdown).
func initialize(cfg *config.Config, logger *slog.Logger) (http.Handler, *sql.DB, error) {
	// Ensure required directories exist (these may live on a slow network mount).
	if err := os.MkdirAll(cfg.FaviconCacheDir, 0o755); err != nil {
		return nil, nil, err
	}
	if cfg.DatabasePath != ":memory:" {
		if dir := filepath.Dir(cfg.DatabasePath); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, nil, err
			}
		}
	}

	db, err := database.OpenWithOptions(cfg.DatabasePath, database.Options{
		JournalMode:      cfg.SQLiteJournalMode,
		Synchronous:      cfg.SQLiteSynchronous,
		SingleConnection: cfg.SQLiteSingleConnection,
	})
	if err != nil {
		return nil, nil, err
	}

	// Repositories.
	folderRepo := repository.NewFolderRepository(db)
	tagRepo := repository.NewTagRepository(db)
	bookmarkRepo := repository.NewBookmarkRepository(db, folderRepo)

	// Services.
	faviconSvc := service.NewFaviconServiceWithBudget(cfg.FaviconCacheDir, cfg.FaviconTimeout, logger)
	metadataSvc := service.NewMetadataService()
	firefoxSvc := service.NewFirefoxService(db, bookmarkRepo, folderRepo, tagRepo, faviconSvc)

	// Templates (embedded).
	tmplFS, err := fs.Sub(web.Files, ".")
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	h, err := handler.New(handler.Deps{
		Config:    cfg,
		Logger:    logger,
		Templates: tmplFS,
		Bookmarks: bookmarkRepo,
		Folders:   folderRepo,
		Tags:      tagRepo,
		Favicons:  faviconSvc,
		Metadata:  metadataSvc,
		Firefox:   firefoxSvc,
	})
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	// Static assets: prefer on-disk favicons dir (writable cache) with an
	// overlay for embedded assets.
	staticFS := staticFileSystem(cfg.FaviconCacheDir)
	mux := h.Routes(staticFS)

	limiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.RateLimitWindow)
	root := middleware.Chain(mux,
		middleware.SecurityHeaders,
		limiter.Middleware,
		middleware.BasicAuth(cfg.AuthUsername, cfg.AuthPassword),
	)

	return root, db, nil
}
