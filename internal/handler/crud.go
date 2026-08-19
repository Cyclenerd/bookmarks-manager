package handler

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/Cyclenerd/bookmarks-manager/internal/repository"
)

// --- Bookmarks ---

func (h *Handler) addBookmarkForm(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	folderID := q.Get("folder")

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

	data := bookmarkFormData{
		Folders:      folders,
		Tags:         tags,
		PrefillURL:   q.Get("url"),
		PrefillTitle: q.Get("title"),
		ReturnURL:    referrerOr(r, "/"),
		SelectedTags: map[string]bool{},
	}
	if folderID != "" {
		data.CurrentFolder, err = h.folders.Get(folderID)
		if err != nil {
			h.serverError(w, err)
			return
		}
	}
	h.renderer.render(w, http.StatusOK, "bookmark_form.html", data)
}

func (h *Handler) editBookmarkForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	bookmark, err := h.bookmarks.Get(id)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if bookmark == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
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
	selected := make(map[string]bool, len(bookmark.Tags))
	for _, t := range bookmark.Tags {
		selected[t.ID] = true
	}
	h.renderer.render(w, http.StatusOK, "bookmark_form.html", bookmarkFormData{
		Bookmark:     bookmark,
		Folders:      folders,
		Tags:         tags,
		ReturnURL:    referrerOr(r, "/"),
		SelectedTags: selected,
	})
}

func (h *Handler) saveBookmark(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	bookmarkID := r.FormValue("bookmark_id")
	rawURL := r.FormValue("url")
	title := r.FormValue("title")
	description := r.FormValue("description")
	folderID := r.FormValue("folder_id")
	tagIDs := r.Form["tag_ids"]
	returnURL := r.FormValue("return_url")

	var folderPtr *string
	if folderID != "" {
		folderPtr = &folderID
	}

	favicon := h.favicons.Download(rawURL)

	var err error
	if bookmarkID != "" {
		err = h.bookmarks.Update(bookmarkID, rawURL, title, description, folderPtr, tagIDs, favicon)
	} else {
		_, err = h.bookmarks.Create(rawURL, title, description, folderPtr, tagIDs, favicon)
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, redirectTarget(returnURL, "/"), http.StatusFound)
}

func (h *Handler) deleteBookmark(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = r.ParseForm()
	returnURL := r.FormValue("return_url")
	if err := h.bookmarks.Delete(id); err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, redirectTarget(returnURL, referrerOr(r, "/")), http.StatusFound)
}

func (h *Handler) togglePin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.bookmarks.TogglePin(id); err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, referrerOr(r, "/"), http.StatusFound)
}

// --- Folders ---

func (h *Handler) addFolderForm(w http.ResponseWriter, r *http.Request) {
	parentID := r.URL.Query().Get("parent")
	folders, err := h.folders.GetHierarchy()
	if err != nil {
		h.serverError(w, err)
		return
	}
	data := folderFormData{Folders: folders, ReturnURL: referrerOr(r, "/")}
	if parentID != "" {
		data.ParentFolder, err = h.folders.Get(parentID)
		if err != nil {
			h.serverError(w, err)
			return
		}
	}
	h.renderer.render(w, http.StatusOK, "folder_form.html", data)
}

func (h *Handler) editFolderForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	folder, err := h.folders.Get(id)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if folder == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	folders, err := h.folders.GetHierarchy()
	if err != nil {
		h.serverError(w, err)
		return
	}
	h.renderer.render(w, http.StatusOK, "folder_form.html", folderFormData{
		Folder:    folder,
		Folders:   folders,
		ReturnURL: referrerOr(r, "/"),
	})
}

func (h *Handler) saveFolder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	folderID := r.FormValue("folder_id")
	name := r.FormValue("name")
	parentID := r.FormValue("parent_id")
	returnURL := r.FormValue("return_url")

	var parentPtr *string
	if parentID != "" {
		parentPtr = &parentID
	}

	var err error
	if folderID != "" {
		err = h.folders.Update(folderID, name, parentPtr)
	} else {
		_, err = h.folders.Create(name, parentPtr)
	}
	if err != nil {
		if errors.Is(err, repository.ErrCircularFolder) {
			h.renderError(w, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, redirectTarget(returnURL, "/"), http.StatusFound)
}

func (h *Handler) deleteFolder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.folders.Delete(id); err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// --- Tags ---

func (h *Handler) addTagForm(w http.ResponseWriter, r *http.Request) {
	h.renderer.render(w, http.StatusOK, "tag_form.html", tagFormData{ReturnURL: referrerOr(r, "/")})
}

func (h *Handler) editTagForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tag, err := h.tags.Get(id)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if tag == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.renderer.render(w, http.StatusOK, "tag_form.html", tagFormData{
		Tag:       tag,
		ReturnURL: referrerOr(r, "/"),
	})
}

func (h *Handler) saveTag(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	tagID := r.FormValue("tag_id")
	name := r.FormValue("name")
	returnURL := r.FormValue("return_url")

	var err error
	if tagID != "" {
		err = h.tags.Update(tagID, name)
	} else {
		_, err = h.tags.Create(name)
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, redirectTarget(returnURL, "/"), http.StatusFound)
}

func (h *Handler) deleteTag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.tags.Delete(id); err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// --- shared redirect helpers ---

// referrerOr returns the Referer header or a fallback.
func referrerOr(r *http.Request, fallback string) string {
	if ref := r.Referer(); ref != "" {
		return ref
	}
	return fallback
}

// redirectTarget prefers primary when set, else the fallback.
func redirectTarget(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}
