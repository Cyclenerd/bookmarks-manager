package handler

import (
	"net/url"

	"github.com/Cyclenerd/bookmarks-manager/internal/models"
)

// pageData is the view model for the index page.
type pageData struct {
	Bookmarks     []models.Bookmark
	Folders       []*models.Folder
	Tags          []models.Tag
	CurrentFolder *models.Folder
	CurrentTag    *models.Tag
	IsUnfiled     bool
	SortBy        string
	SortOrder     string
	Page          int
	TotalPages    int
	Total         int
	BaseURL       string

	// filter parameters used to build links
	folderParam string
	tagParam    string
	search      string
}

// CurrentFolderParam returns the raw folder query parameter for nav links.
func (p pageData) CurrentFolderParam() string { return p.folderParam }

// FilterQuery returns the URL-encoded folder/tag/search filters (without sort).
func (p pageData) FilterQuery() string {
	v := url.Values{}
	if p.folderParam != "" {
		v.Set("folder", p.folderParam)
	}
	if p.tagParam != "" {
		v.Set("tag", p.tagParam)
	}
	if p.search != "" {
		v.Set("search", p.search)
	}
	return v.Encode()
}

// nextOrder toggles asc/desc for the currently-active sort column.
func (p pageData) nextOrder() string {
	if p.SortOrder == "desc" {
		return "asc"
	}
	return "desc"
}

// TitleOrder returns the order to use when clicking the Title sort button.
func (p pageData) TitleOrder() string { return sortLinkOrder(p.SortBy, "title", p.nextOrder()) }

// URLOrder returns the order to use when clicking the URL sort button.
func (p pageData) URLOrder() string { return sortLinkOrder(p.SortBy, "url", p.nextOrder()) }

// DateOrder returns the order to use when clicking the Date sort button.
func (p pageData) DateOrder() string { return sortLinkOrder(p.SortBy, "created_at", p.nextOrder()) }

func sortLinkOrder(current, column, next string) string {
	if current == column {
		return next
	}
	return "asc"
}

// ShowingFrom is the 1-based index of the first bookmark on the page.
func (p pageData) ShowingFrom() int {
	if len(p.Bookmarks) == 0 {
		return 0
	}
	return (p.Page-1)*25 + 1
}

// ShowingTo is the 1-based index of the last bookmark on the page.
func (p pageData) ShowingTo() int {
	return (p.Page-1)*25 + len(p.Bookmarks)
}

// PrevPage / NextPage are clamped page links.
func (p pageData) PrevPage() int { return p.Page - 1 }
func (p pageData) NextPage() int { return p.Page + 1 }

// PageNumbers returns the windowed pagination sequence, using -1 as an ellipsis
// marker, matching the original template's logic.
func (p pageData) PageNumbers() []int {
	return pageWindow(p.Page, p.TotalPages)
}

// searchData is the view model for the search results page.
type searchData struct {
	Bookmarks  []models.Bookmark
	Query      string
	SortBy     string
	SortOrder  string
	Page       int
	TotalPages int
	Total      int
}

// CurrentFolderParam satisfies the base template (no active folder here).
func (searchData) CurrentFolderParam() string { return "" }

func (s searchData) nextOrder() string {
	if s.SortOrder == "desc" {
		return "asc"
	}
	return "desc"
}
func (s searchData) TitleOrder() string { return sortLinkOrder(s.SortBy, "title", s.nextOrder()) }
func (s searchData) URLOrder() string   { return sortLinkOrder(s.SortBy, "url", s.nextOrder()) }
func (s searchData) DateOrder() string  { return sortLinkOrder(s.SortBy, "created_at", s.nextOrder()) }
func (s searchData) ShowingFrom() int {
	if len(s.Bookmarks) == 0 {
		return 0
	}
	return (s.Page-1)*25 + 1
}
func (s searchData) ShowingTo() int     { return (s.Page-1)*25 + len(s.Bookmarks) }
func (s searchData) PrevPage() int      { return s.Page - 1 }
func (s searchData) NextPage() int      { return s.Page + 1 }
func (s searchData) PageNumbers() []int { return pageWindow(s.Page, s.TotalPages) }

// bookmarkFormData is the view model for the add/edit bookmark form.
type bookmarkFormData struct {
	Bookmark      *models.Bookmark
	Folders       []*models.Folder
	Tags          []models.Tag
	CurrentFolder *models.Folder
	PrefillURL    string
	PrefillTitle  string
	ReturnURL     string
	SelectedTags  map[string]bool
}

// CurrentFolderParam satisfies the base template (no active folder here).
func (bookmarkFormData) CurrentFolderParam() string { return "" }

// SelectedFolderID returns the folder that should be pre-selected.
func (d bookmarkFormData) SelectedFolderID() string {
	if d.Bookmark != nil && d.Bookmark.FolderID != nil {
		return *d.Bookmark.FolderID
	}
	if d.Bookmark == nil && d.CurrentFolder != nil {
		return d.CurrentFolder.ID
	}
	return ""
}

// folderFormData is the view model for the add/edit folder form.
type folderFormData struct {
	Folder       *models.Folder
	Folders      []*models.Folder
	ParentFolder *models.Folder
	ReturnURL    string
}

func (folderFormData) CurrentFolderParam() string { return "" }

func (d folderFormData) SelectedFolderID() string {
	if d.Folder != nil && d.Folder.ParentID != nil {
		return *d.Folder.ParentID
	}
	if d.Folder == nil && d.ParentFolder != nil {
		return d.ParentFolder.ID
	}
	return ""
}

// tagFormData is the view model for the add/edit tag form.
type tagFormData struct {
	Tag       *models.Tag
	ReturnURL string
}

func (tagFormData) CurrentFolderParam() string { return "" }

// importSuccessData / importErrorData / errorData are simple view models.
type importSuccessData struct {
	Stats models.ImportStats
}

func (importSuccessData) CurrentFolderParam() string { return "" }

type importPageData struct{}

func (importPageData) CurrentFolderParam() string { return "" }

type importErrorData struct {
	Error string
}

func (importErrorData) CurrentFolderParam() string { return "" }

type errorData struct {
	ErrorCode        int
	ErrorName        string
	ErrorDescription string
}

func (errorData) CurrentFolderParam() string { return "" }

// pageWindow reproduces the original pagination window: always show page 1 and
// the last page, plus a window of +/-2 around the current page, inserting a
// single ellipsis (-1) where numbers are skipped.
func pageWindow(page, total int) []int {
	var out []int
	prevPrinted := 0
	for p := 1; p <= total; p++ {
		show := p == 1 || p == total || (p >= page-2 && p <= page+2)
		if show {
			if prevPrinted != 0 && p-prevPrinted > 1 {
				out = append(out, -1)
			}
			out = append(out, p)
			prevPrinted = p
		}
	}
	return out
}
