package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Cyclenerd/bookmarks-manager/internal/models"
	"github.com/Cyclenerd/bookmarks-manager/internal/repository"
)

const perPage = 25

// index renders the main bookmark listing.
func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	folderID := q.Get("folder")
	tagID := q.Get("tag")
	search := q.Get("search")
	sortBy := def(q.Get("sort"), "created_at")
	sortOrder := def(q.Get("order"), "desc")
	page := atoiDefault(q.Get("page"), 1)

	// A non-empty search on the index redirects to the dedicated search page.
	if search != "" {
		http.Redirect(w, r, "/search?q="+urlQueryEscape(search), http.StatusFound)
		return
	}

	isUnfiled := folderID == "unfiled"

	folders, err := h.folders.GetHierarchy()
	if err != nil {
		h.serverError(w, err)
		return
	}
	tags, err := h.tags.GetAll()
	if err != nil {
		h.serverError(w, err)
		return
	}

	result, err := h.bookmarks.GetAll(repository.ListOptions{
		FolderID:          folderID,
		TagID:             tagID,
		SortBy:            sortBy,
		SortOrder:         sortOrder,
		IncludeSubfolders: false,
		Page:              page,
		PerPage:           perPage,
	})
	if err != nil {
		h.serverError(w, err)
		return
	}

	var currentFolder *models.Folder
	var currentTag *models.Tag
	if folderID != "" && !isUnfiled {
		currentFolder, err = h.folders.Get(folderID)
		if err != nil {
			h.serverError(w, err)
			return
		}
	}
	if tagID != "" {
		currentTag, err = h.tags.Get(tagID)
		if err != nil {
			h.serverError(w, err)
			return
		}
	}

	h.renderer.render(w, http.StatusOK, "index.html", pageData{
		Bookmarks:     result.Bookmarks,
		Folders:       folders,
		Tags:          tags,
		CurrentFolder: currentFolder,
		CurrentTag:    currentTag,
		IsUnfiled:     isUnfiled,
		SortBy:        sortBy,
		SortOrder:     sortOrder,
		Page:          result.Page,
		TotalPages:    result.TotalPages,
		Total:         result.Total,
		BaseURL:       baseURL(r),
		folderParam:   folderID,
		tagParam:      tagID,
		search:        search,
	})
}

// searchPage renders the full-text search results page.
func (h *Handler) searchPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("q")
	sortBy := def(q.Get("sort"), "created_at")
	sortOrder := def(q.Get("order"), "desc")
	page := atoiDefault(q.Get("page"), 1)

	result := &models.BookmarkPage{Page: 1}
	if query != "" {
		var err error
		result, err = h.bookmarks.GetAll(repository.ListOptions{
			Search:    query,
			SortBy:    sortBy,
			SortOrder: sortOrder,
			Page:      page,
			PerPage:   perPage,
		})
		if err != nil {
			h.serverError(w, err)
			return
		}
	}

	h.renderer.render(w, http.StatusOK, "search_results.html", searchData{
		Bookmarks:  result.Bookmarks,
		Query:      query,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		Page:       result.Page,
		TotalPages: result.TotalPages,
		Total:      result.Total,
	})
}

// fetchMetadata is the JSON API for auto-filling a bookmark title.
func (h *Handler) fetchMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false, "error": "Content-Type must be application/json",
		})
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false, "error": "URL is required",
		})
		return
	}
	meta := h.metadata.FetchTitle(body.URL)
	writeJSON(w, http.StatusOK, meta)
}

// searchAPI powers the live-search dropdown (max 10 results).
func (h *Handler) searchAPI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{"bookmarks": []any{}})
		return
	}
	result, err := h.bookmarks.GetAll(repository.ListOptions{
		Search: query, SortBy: "created_at", SortOrder: "desc", Page: 1, PerPage: 10,
	})
	if err != nil {
		h.serverError(w, err)
		return
	}

	type item struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		URL        string `json:"url"`
		FolderName string `json:"folder_name"`
		Favicon    string `json:"favicon"`
	}
	items := make([]item, 0, len(result.Bookmarks))
	for _, b := range result.Bookmarks {
		items = append(items, item{
			ID: b.ID, Title: b.Title, URL: b.URL, FolderName: b.FolderName, Favicon: b.Favicon,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"bookmarks": items})
}

func (h *Handler) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("User-agent: *\nDisallow: /"))
}

func (h *Handler) faviconIco(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/img/favicon.ico", http.StatusMovedPermanently)
}

// --- small helpers ---

func def(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func atoiDefault(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
