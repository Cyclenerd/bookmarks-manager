package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/Cyclenerd/bookmarks-manager/internal/models"
)

// pageTemplates lists the content templates that each get combined with the
// base layout and shared partials into their own template set.
var pageTemplates = []string{
	"index.html",
	"search_results.html",
	"bookmark_form.html",
	"folder_form.html",
	"tag_form.html",
	"import.html",
	"import_success.html",
	"import_error.html",
	"error.html",
}

// renderer holds the parsed template sets keyed by page name.
type renderer struct {
	sets map[string]*template.Template
}

// newRenderer parses the shared layout (base.html + partials.html) exactly
// once, then Clone()s it for each page and layers only that page's template on
// top. This avoids re-parsing the shared files N times, which is the dominant
// template cost during a cold start.
func newRenderer(fsys fs.FS) (*renderer, error) {
	funcs := templateFuncs()

	// Parse the shared layout a single time.
	shared, err := template.New("base").Funcs(funcs).ParseFS(fsys,
		"templates/base.html",
		"templates/partials.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse shared layout: %w", err)
	}

	sets := make(map[string]*template.Template, len(pageTemplates))
	for _, page := range pageTemplates {
		// Clone the already-parsed shared layout instead of re-parsing it.
		t, err := shared.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layout for %s: %w", page, err)
		}
		if _, err := t.ParseFS(fsys, "templates/"+page); err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		sets[page] = t
	}
	return &renderer{sets: sets}, nil
}

// render executes the named page with data and writes it, buffering first so a
// template error does not produce a partially-written response.
func (r *renderer) render(w http.ResponseWriter, status int, page string, data any) {
	set, ok := r.sets[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := set.ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// --- recursive template helper contexts ---

// navCtx carries state for rendering a folder navigation subtree.
type navCtx struct {
	Folder        *models.Folder
	Level         int
	CurrentFolder *models.Folder
	SortBy        string
	SortOrder     string
}

// optCtx carries state for rendering a <select> folder option subtree.
type optCtx struct {
	Folder     *models.Folder
	Level      int
	SelectedID string
	// ExcludeID and its descendants are still rendered (parity with the
	// original), selection is by SelectedID only.
}

// Selected reports whether this folder option should be marked selected.
func (o optCtx) Selected() bool {
	return o.SelectedID != "" && o.Folder.ID == o.SelectedID
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// iter returns a slice of length n for range-based repetition.
		"iter": func(n int) []struct{} {
			return make([]struct{}, n)
		},
		// navRoot starts a folder-nav subtree from a page context.
		"navRoot": func(page pageData, f *models.Folder) navCtx {
			return navCtx{Folder: f, Level: 0, CurrentFolder: page.CurrentFolder, SortBy: page.SortBy, SortOrder: page.SortOrder}
		},
		// navChild descends one level in a folder-nav subtree.
		"navChild": func(parent navCtx, f *models.Folder) navCtx {
			return navCtx{Folder: f, Level: parent.Level + 1, CurrentFolder: parent.CurrentFolder, SortBy: parent.SortBy, SortOrder: parent.SortOrder}
		},
		// optRoot starts a folder <option> subtree; the selected ID is read
		// from the page context via the SelectedFolderID accessor.
		"optRoot": func(page selectedFolderProvider, f *models.Folder) optCtx {
			return optCtx{Folder: f, Level: 0, SelectedID: page.SelectedFolderID()}
		},
		// optChild descends one level in a folder <option> subtree.
		"optChild": func(parent optCtx, f *models.Folder) optCtx {
			return optCtx{Folder: f, Level: parent.Level + 1, SelectedID: parent.SelectedID}
		},
	}
}

// selectedFolderProvider is implemented by form page data that has a currently
// selected folder for a <select>.
type selectedFolderProvider interface {
	SelectedFolderID() string
}
