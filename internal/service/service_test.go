package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Cyclenerd/bookmarks-manager/internal/database"
	"github.com/Cyclenerd/bookmarks-manager/internal/repository"
)

func TestExtractTitlePrefersOpenGraph(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head>
			<meta property="og:title" content="OG Title">
			<title>Plain Title</title>
		</head><body></body></html>`))
	}))
	defer srv.Close()

	meta := NewMetadataService().FetchTitle(srv.URL)
	if !meta.Success {
		t.Fatalf("expected success, got %+v", meta)
	}
	if meta.Title != "OG Title" {
		t.Errorf("expected og:title preference, got %q", meta.Title)
	}
}

func TestExtractTitleFallsBackToTitleTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>  Just Title  </title></head></html>`))
	}))
	defer srv.Close()

	meta := NewMetadataService().FetchTitle(srv.URL)
	if meta.Title != "Just Title" {
		t.Errorf("expected trimmed title tag, got %q", meta.Title)
	}
}

func TestFirefoxRoundTrip(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	folders := repository.NewFolderRepository(db)
	tags := repository.NewTagRepository(db)
	bookmarks := repository.NewBookmarkRepository(db, folders)
	// Favicon service pointed at a no-op dir; downloads fail fast for invalid hosts.
	fav := NewFaviconService(t.TempDir(), testLogger())
	ff := NewFirefoxService(db, bookmarks, folders, tags, fav)

	fid, _ := folders.Create("Work", nil)
	_, _ = bookmarks.Create("https://a.com", "Alpha", "hello", &fid, nil, "")

	exported, err := ff.Export()
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(string(exported), "root________") {
		t.Errorf("export missing root guid")
	}
	if !strings.Contains(string(exported), "Alpha") {
		t.Errorf("export missing bookmark")
	}

	node, err := ff.Parse(exported)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	stats, err := ff.Import(node)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if stats.Bookmarks != 1 || stats.Folders != 1 {
		t.Errorf("unexpected import stats: %+v", stats)
	}
}

func TestFirefoxParseInvalid(t *testing.T) {
	ff := &FirefoxService{}
	if _, err := ff.Parse([]byte("not json{{")); err == nil {
		t.Errorf("expected parse error on invalid json")
	}
}
