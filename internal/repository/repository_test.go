package repository

import (
	"errors"
	"testing"

	"github.com/Cyclenerd/bookmarks-manager/internal/database"
)

func newTestRepos(t *testing.T) (*BookmarkRepository, *FolderRepository, *TagRepository) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	folders := NewFolderRepository(db)
	tags := NewTagRepository(db)
	bookmarks := NewBookmarkRepository(db, folders)
	return bookmarks, folders, tags
}

func TestBookmarkCRUD(t *testing.T) {
	bookmarks, _, _ := newTestRepos(t)

	id, err := bookmarks.Create("https://example.com", "Example", "desc", nil, nil, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := bookmarks.Get(id)
	if err != nil || got == nil {
		t.Fatalf("get: %v got=%v", err, got)
	}
	if got.Title != "Example" || got.URL != "https://example.com" {
		t.Errorf("unexpected bookmark %+v", got)
	}

	if err := bookmarks.Update(id, "https://example.org", "Updated", "d2", nil, nil, ""); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = bookmarks.Get(id)
	if got.Title != "Updated" || got.URL != "https://example.org" {
		t.Errorf("update not applied: %+v", got)
	}

	pinned, err := bookmarks.TogglePin(id)
	if err != nil || !pinned {
		t.Fatalf("toggle pin: %v pinned=%v", err, pinned)
	}

	if err := bookmarks.Delete(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ = bookmarks.Get(id)
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

func TestBookmarkFilteringAndPinnedOrder(t *testing.T) {
	bookmarks, folders, tags := newTestRepos(t)

	fid, _ := folders.Create("Work", nil)
	tid, _ := tags.Create("go")

	_, _ = bookmarks.Create("https://a.com", "Alpha", "", &fid, []string{tid}, "")
	_, _ = bookmarks.Create("https://b.com", "Bravo", "", nil, nil, "")
	pinnedID, _ := bookmarks.Create("https://c.com", "Charlie", "", nil, nil, "")
	_, _ = bookmarks.TogglePin(pinnedID)

	// All bookmarks: pinned must come first.
	page, err := bookmarks.GetAll(ListOptions{SortBy: "title", SortOrder: "asc", PerPage: 25, Page: 1})
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if page.Total != 3 {
		t.Fatalf("expected 3 total, got %d", page.Total)
	}
	if page.Bookmarks[0].Title != "Charlie" {
		t.Errorf("pinned bookmark should be first, got %s", page.Bookmarks[0].Title)
	}

	// Filter by folder.
	page, _ = bookmarks.GetAll(ListOptions{FolderID: fid, PerPage: 25, Page: 1})
	if page.Total != 1 || page.Bookmarks[0].Title != "Alpha" {
		t.Errorf("folder filter failed: %+v", page)
	}

	// Filter by unfiled.
	page, _ = bookmarks.GetAll(ListOptions{FolderID: "unfiled", PerPage: 25, Page: 1})
	if page.Total != 2 {
		t.Errorf("unfiled filter expected 2, got %d", page.Total)
	}

	// Filter by tag.
	page, _ = bookmarks.GetAll(ListOptions{TagID: tid, PerPage: 25, Page: 1})
	if page.Total != 1 {
		t.Errorf("tag filter expected 1, got %d", page.Total)
	}

	// Search.
	page, _ = bookmarks.GetAll(ListOptions{Search: "Brav", PerPage: 25, Page: 1})
	if page.Total != 1 || page.Bookmarks[0].Title != "Bravo" {
		t.Errorf("search failed: %+v", page)
	}
}

func TestFolderHierarchyAndCircular(t *testing.T) {
	_, folders, _ := newTestRepos(t)

	parent, _ := folders.Create("Parent", nil)
	child, _ := folders.Create("Child", &parent)

	tree, err := folders.GetHierarchy()
	if err != nil {
		t.Fatalf("hierarchy: %v", err)
	}
	if len(tree) != 1 || tree[0].Name != "Parent" || len(tree[0].Children) != 1 {
		t.Fatalf("unexpected tree: %+v", tree)
	}

	// Moving parent into its own child must fail.
	err = folders.Update(parent, "Parent", &child)
	if !errors.Is(err, ErrCircularFolder) {
		t.Errorf("expected ErrCircularFolder, got %v", err)
	}

	ids, _ := folders.DescendantIDs(parent)
	if len(ids) != 2 {
		t.Errorf("expected 2 descendant ids, got %d", len(ids))
	}
}

func TestDeleteFolderDetachesBookmarks(t *testing.T) {
	bookmarks, folders, _ := newTestRepos(t)

	fid, _ := folders.Create("Temp", nil)
	bid, _ := bookmarks.Create("https://x.com", "X", "", &fid, nil, "")

	if err := folders.Delete(fid); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	got, _ := bookmarks.Get(bid)
	if got == nil || got.FolderID != nil {
		t.Errorf("expected bookmark detached, got %+v", got)
	}
}

func TestTagUnique(t *testing.T) {
	_, _, tags := newTestRepos(t)

	if _, err := tags.Create("dup"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := tags.Create("dup"); err == nil {
		t.Errorf("expected unique constraint error on duplicate tag")
	}

	found, _ := tags.FindByName("dup")
	if found == nil {
		t.Errorf("FindByName should locate existing tag")
	}
}
