// Package models defines the domain types shared across the application.
package models

// Tag represents a single label that can be attached to bookmarks.
type Tag struct {
	ID            string
	Name          string
	CreatedAt     string
	BookmarkCount int
}

// Folder represents a hierarchical container for bookmarks.
type Folder struct {
	ID             string
	Name           string
	ParentID       *string
	CreatedAt      string
	BookmarkCount  int
	SubfolderCount int
	// Children holds nested folders when a hierarchy is built.
	Children []*Folder
	// ParentChain holds ancestor folders from root to immediate parent.
	ParentChain []*Folder
}

// Bookmark represents a saved link with optional folder, tags and favicon.
type Bookmark struct {
	ID          string
	URL         string
	Title       string
	Description string
	Favicon     string
	FolderID    *string
	FolderName  string
	Pinned      bool
	CreatedAt   string
	Tags        []Tag
}

// BookmarkPage is a paginated result set of bookmarks.
type BookmarkPage struct {
	Bookmarks  []Bookmark
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

// ImportStats reports the outcome of a Firefox import operation.
type ImportStats struct {
	Bookmarks int
	Folders   int
	Tags      int
}
