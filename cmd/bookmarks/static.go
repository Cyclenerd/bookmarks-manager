package main

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/Cyclenerd/bookmarks-manager/web"
)

// staticFileSystem serves static assets, resolving "/favicons/..." from the
// on-disk favicon cache (which is writable and may live on a mounted volume)
// and everything else from the embedded assets.
func staticFileSystem(faviconDir string) http.FileSystem {
	embedded, _ := fs.Sub(web.Files, "static")
	return &overlayFS{
		embedded:   http.FS(embedded),
		faviconDir: http.Dir(faviconDir),
	}
}

type overlayFS struct {
	embedded   http.FileSystem
	faviconDir http.FileSystem
}

// Open serves favicons from the on-disk cache and all other paths from the
// embedded filesystem.
func (o *overlayFS) Open(name string) (http.File, error) {
	trimmed := strings.TrimPrefix(name, "/")
	if strings.HasPrefix(trimmed, "favicons/") {
		return o.faviconDir.Open(strings.TrimPrefix(trimmed, "favicons/"))
	}
	return o.embedded.Open(name)
}
