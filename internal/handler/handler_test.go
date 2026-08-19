package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Cyclenerd/bookmarks-manager/internal/config"
	"github.com/Cyclenerd/bookmarks-manager/internal/database"
	"github.com/Cyclenerd/bookmarks-manager/internal/repository"
	"github.com/Cyclenerd/bookmarks-manager/internal/service"
	"github.com/Cyclenerd/bookmarks-manager/web"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	folders := repository.NewFolderRepository(db)
	tags := repository.NewTagRepository(db)
	bookmarks := repository.NewBookmarkRepository(db, folders)
	fav := service.NewFaviconService(t.TempDir(), testLogger())

	h, err := New(Deps{
		Config:    config.Load(),
		Logger:    testLogger(),
		Templates: web.Files,
		Bookmarks: bookmarks,
		Folders:   folders,
		Tags:      tags,
		Favicons:  fav,
		Metadata:  service.NewMetadataService(),
		Firefox:   service.NewFirefoxService(db, bookmarks, folders, tags, fav),
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return h.Routes(http.Dir(t.TempDir()))
}

func TestIndexRenders(t *testing.T) {
	srv := httptest.NewServer(newTestServer(t))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "All Bookmarks") {
		t.Errorf("index missing expected content")
	}
}

func TestCreateAndListBookmark(t *testing.T) {
	handler := newTestServer(t)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	form := strings.NewReader("url=https://example.invalid/p&title=Hello+World")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/bookmark/save", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Prevent redirect follow so we can assert the 302.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	resp, _ = http.Get(srv.URL + "/")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "Hello World") {
		t.Errorf("created bookmark not shown on index")
	}
}

func TestSearchAPIMinLength(t *testing.T) {
	srv := httptest.NewServer(newTestServer(t))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/search?q=a")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `"bookmarks":[]`) {
		t.Errorf("expected empty results for short query, got %s", body)
	}
}
