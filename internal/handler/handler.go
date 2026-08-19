// Package handler wires HTTP routes to repositories, services and templates.
package handler

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/Cyclenerd/bookmarks-manager/internal/config"
	"github.com/Cyclenerd/bookmarks-manager/internal/repository"
	"github.com/Cyclenerd/bookmarks-manager/internal/service"
)

// Handler holds all dependencies needed to serve the application.
type Handler struct {
	cfg       *config.Config
	logger    *slog.Logger
	renderer  *renderer
	bookmarks *repository.BookmarkRepository
	folders   *repository.FolderRepository
	tags      *repository.TagRepository
	favicons  *service.FaviconService
	metadata  *service.MetadataService
	firefox   *service.FirefoxService
}

// Deps bundles the constructed dependencies for New.
type Deps struct {
	Config    *config.Config
	Logger    *slog.Logger
	Templates fs.FS
	Bookmarks *repository.BookmarkRepository
	Folders   *repository.FolderRepository
	Tags      *repository.TagRepository
	Favicons  *service.FaviconService
	Metadata  *service.MetadataService
	Firefox   *service.FirefoxService
}

// New constructs a Handler, parsing templates from the provided filesystem.
func New(d Deps) (*Handler, error) {
	r, err := newRenderer(d.Templates)
	if err != nil {
		return nil, err
	}
	return &Handler{
		cfg:       d.Config,
		logger:    d.Logger,
		renderer:  r,
		bookmarks: d.Bookmarks,
		folders:   d.Folders,
		tags:      d.Tags,
		favicons:  d.Favicons,
		metadata:  d.Metadata,
		firefox:   d.Firefox,
	}, nil
}

// Routes registers all application routes on a new ServeMux and returns it.
// staticFS should be rooted so that "/static/..." maps correctly.
func (h *Handler) Routes(staticFS http.FileSystem) *http.ServeMux {
	mux := http.NewServeMux()

	// Static assets and unauthenticated files.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(staticFS)))
	mux.HandleFunc("GET /robots.txt", h.robots)
	mux.HandleFunc("GET /favicon.ico", h.faviconIco)

	// Pages.
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("GET /search", h.searchPage)

	// APIs.
	mux.HandleFunc("POST /api/fetch-metadata", h.fetchMetadata)
	mux.HandleFunc("GET /api/search", h.searchAPI)

	// Bookmarks.
	mux.HandleFunc("GET /bookmark/add", h.addBookmarkForm)
	mux.HandleFunc("GET /bookmark/{id}/edit", h.editBookmarkForm)
	mux.HandleFunc("POST /bookmark/save", h.saveBookmark)
	mux.HandleFunc("POST /bookmark/{id}/delete", h.deleteBookmark)
	mux.HandleFunc("POST /bookmark/{id}/toggle-pin", h.togglePin)

	// Folders.
	mux.HandleFunc("GET /folder/add", h.addFolderForm)
	mux.HandleFunc("GET /folder/{id}/edit", h.editFolderForm)
	mux.HandleFunc("POST /folder/save", h.saveFolder)
	mux.HandleFunc("POST /folder/{id}/delete", h.deleteFolder)

	// Tags.
	mux.HandleFunc("GET /tag/add", h.addTagForm)
	mux.HandleFunc("GET /tag/{id}/edit", h.editTagForm)
	mux.HandleFunc("POST /tag/save", h.saveTag)
	mux.HandleFunc("POST /tag/{id}/delete", h.deleteTag)

	// Import / export.
	mux.HandleFunc("GET /import", h.importPage)
	mux.HandleFunc("POST /import/firefox", h.importFirefox)
	mux.HandleFunc("GET /export/firefox", h.exportFirefox)

	return mux
}

// serverError logs err and renders a 500 error page.
func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.logger.Error("server error", "err", err)
	h.renderError(w, http.StatusInternalServerError, "Internal Server Error",
		"The server encountered an internal error and was unable to complete your request.")
}

// renderError renders the shared error page with the given status.
func (h *Handler) renderError(w http.ResponseWriter, code int, name, description string) {
	h.renderer.render(w, code, "error.html", errorData{
		ErrorCode:        code,
		ErrorName:        name,
		ErrorDescription: description,
	})
}
