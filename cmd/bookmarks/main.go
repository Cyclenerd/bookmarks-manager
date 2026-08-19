// Command bookmarks is the entrypoint for the self-hosted bookmark manager.
package main

import (
	"context"
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
	cfg := config.Load()

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	// Ensure the favicon cache directory exists.
	if err := os.MkdirAll(cfg.FaviconCacheDir, 0o755); err != nil {
		return err
	}
	// Ensure the database directory exists.
	if cfg.DatabasePath != ":memory:" {
		if dir := filepath.Dir(cfg.DatabasePath); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Repositories.
	folderRepo := repository.NewFolderRepository(db)
	tagRepo := repository.NewTagRepository(db)
	bookmarkRepo := repository.NewBookmarkRepository(db, folderRepo)

	// Services.
	faviconSvc := service.NewFaviconService(cfg.FaviconCacheDir, logger)
	metadataSvc := service.NewMetadataService()
	firefoxSvc := service.NewFirefoxService(db, bookmarkRepo, folderRepo, tagRepo, faviconSvc)

	// Templates (embedded).
	tmplFS, err := fs.Sub(web.Files, ".")
	if err != nil {
		return err
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
		return err
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

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("shutting down")
	return srv.Shutdown(ctx)
}
