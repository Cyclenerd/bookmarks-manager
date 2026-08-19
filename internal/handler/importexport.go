package handler

import (
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

func (h *Handler) importPage(w http.ResponseWriter, r *http.Request) {
	h.renderer.render(w, http.StatusOK, "import.html", importPageData{})
}

func (h *Handler) importFirefox(w http.ResponseWriter, r *http.Request) {
	// Enforce the 128 KB upload limit.
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxContentLength)
	if err := r.ParseMultipartForm(h.cfg.MaxContentLength); err != nil {
		h.renderer.render(w, http.StatusOK, "import_error.html", importErrorData{
			Error: "File too large (max. 128 KB) or malformed upload.",
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, "/import", http.StatusFound)
		return
	}
	defer file.Close()

	if header.Filename == "" {
		http.Redirect(w, r, "/import", http.StatusFound)
		return
	}
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".json") {
		h.renderer.render(w, http.StatusOK, "import_error.html", importErrorData{
			Error: "Only JSON files are allowed",
		})
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		h.renderer.render(w, http.StatusOK, "import_error.html", importErrorData{Error: err.Error()})
		return
	}
	if !utf8.Valid(content) {
		h.renderer.render(w, http.StatusOK, "import_error.html", importErrorData{
			Error: "Invalid file encoding. Must be UTF-8.",
		})
		return
	}

	node, err := h.firefox.Parse(content)
	if err != nil {
		h.renderer.render(w, http.StatusOK, "import_error.html", importErrorData{Error: err.Error()})
		return
	}
	stats, err := h.firefox.Import(node)
	if err != nil {
		h.renderer.render(w, http.StatusOK, "import_error.html", importErrorData{Error: err.Error()})
		return
	}
	h.renderer.render(w, http.StatusOK, "import_success.html", importSuccessData{Stats: stats})
}

func (h *Handler) exportFirefox(w http.ResponseWriter, r *http.Request) {
	data, err := h.firefox.Export()
	if err != nil {
		h.serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=bookmarks.json")
	_, _ = w.Write(data)
}
