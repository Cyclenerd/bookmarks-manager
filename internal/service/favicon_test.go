package service

import (
	"bytes"
	"compress/gzip"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ico "github.com/biessek/golang-ico"
)

// pngBytes returns an encoded PNG of the given size filled with one colour.
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// icoBytes returns an encoded ICO of the given size.
func icoBytes(t *testing.T, size int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 50, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := ico.Encode(&buf, img); err != nil {
		t.Fatalf("encode ico: %v", err)
	}
	return buf.Bytes()
}

// savedFavicon reads back the cached favicon and returns its decoded image.
func savedFavicon(t *testing.T, dir, rel string) image.Image {
	t.Helper()
	if !strings.HasPrefix(rel, "favicons/") {
		t.Fatalf("unexpected favicon path %q", rel)
	}
	f, err := os.Open(filepath.Join(dir, strings.TrimPrefix(rel, "favicons/")))
	if err != nil {
		t.Fatalf("open saved favicon: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode saved favicon: %v", err)
	}
	return img
}

// TestDownloadFromGzippedHTML is the regression test for the primary bug:
// pages served with gzip must still be parsed for their declared icons.
func TestDownloadFromGzippedHTML(t *testing.T) {
	icon := pngBytes(t, 32, 32)

	mux := http.NewServeMux()
	mux.HandleFunc("/icon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(icon)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			t.Errorf("Go transport should advertise gzip when Accept-Encoding is unset; got %q",
				r.Header.Get("Accept-Encoding"))
		}
		body := `<html><head><link rel="icon" href="/icon.png"></head><body>hi</body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte(body))
		_ = gz.Close()
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	svc := NewFaviconService(dir, testLogger())
	got := svc.Download(srv.URL)
	if got == "" {
		t.Fatal("expected favicon to be found on gzipped page")
	}
	savedFavicon(t, dir, got) // must decode
}

// TestDownloadFaviconICOWrongContentType ensures we sniff bytes rather than
// trusting Content-Type: many servers mislabel favicon.ico.
func TestDownloadFaviconICOWrongContentType(t *testing.T) {
	icon := icoBytes(t, 32)

	mux := http.NewServeMux()
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		// Deliberately wrong content type.
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(icon)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head></head><body>no declared icon</body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	svc := NewFaviconService(dir, testLogger())
	got := svc.Download(srv.URL)
	if got == "" {
		t.Fatal("expected favicon.ico to be found despite wrong Content-Type")
	}
	savedFavicon(t, dir, got)
}

// TestLargeIconIsDownscaled verifies large icons are resized, not rejected.
func TestLargeIconIsDownscaled(t *testing.T) {
	icon := pngBytes(t, 512, 512)

	mux := http.NewServeMux()
	mux.HandleFunc("/big.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(icon)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><link rel="apple-touch-icon" sizes="512x512" href="/big.png"></head></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	svc := NewFaviconService(dir, testLogger())
	got := svc.Download(srv.URL)
	if got == "" {
		t.Fatal("expected large icon to be accepted and downscaled")
	}
	img := savedFavicon(t, dir, got)
	if img.Bounds().Dx() > thumbSize || img.Bounds().Dy() > thumbSize {
		t.Errorf("expected icon downscaled to <=%dpx, got %dx%d", thumbSize, img.Bounds().Dx(), img.Bounds().Dy())
	}
}

// TestIconScoringPrefersLargerAppleTouch verifies preference ordering.
func TestIconScoring(t *testing.T) {
	small := scoreIcon("icon", "16x16", "image/png")
	large := scoreIcon("icon", "32x32", "image/png")
	apple := scoreIcon("apple-touch-icon", "180x180", "image/png")
	svg := scoreIcon("icon", "any", "image/svg+xml")

	if !(large > small) {
		t.Errorf("larger icon should outscore smaller: %d vs %d", large, small)
	}
	if !(apple > large) {
		t.Errorf("apple-touch-icon should outscore a plain 32px icon: %d vs %d", apple, large)
	}
	if !(svg > small) {
		t.Errorf("svg should outscore a tiny raster icon: %d vs %d", svg, small)
	}
}

// TestBestIconChosenFirst ensures the highest-scored declared icon is tried
// first and used.
func TestBestIconChosenFirst(t *testing.T) {
	good := pngBytes(t, 64, 64)

	var smallRequested bool
	mux := http.NewServeMux()
	mux.HandleFunc("/small.png", func(w http.ResponseWriter, r *http.Request) {
		smallRequested = true
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes(t, 16, 16))
	})
	mux.HandleFunc("/apple.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(good)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head>
			<link rel="icon" sizes="16x16" href="/small.png">
			<link rel="apple-touch-icon" sizes="180x180" href="/apple.png">
		</head></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	svc := NewFaviconService(dir, testLogger())
	got := svc.Download(srv.URL)
	if got == "" {
		t.Fatal("expected a favicon")
	}
	if smallRequested {
		t.Errorf("the smaller icon should not have been fetched; the apple-touch-icon should be preferred")
	}
}

// TestDownloadRespectsTotalBudget ensures a slow site that never serves a
// favicon fails fast within the configured budget instead of accumulating a
// separate timeout per candidate URL.
func TestDownloadRespectsTotalBudget(t *testing.T) {
	// Every response hangs until the request context is cancelled.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	budget := 500 * time.Millisecond
	svc := NewFaviconServiceWithBudget(t.TempDir(), budget, testLogger())

	start := time.Now()
	got := svc.Download(srv.URL)
	elapsed := time.Since(start)

	if got != "" {
		t.Errorf("expected no favicon from a hanging server, got %q", got)
	}
	// Allow generous slack for CI, but it must be far below the naive worst
	// case (which would be several multiples of the budget without sharing it).
	if elapsed > 3*budget {
		t.Errorf("download took %v, expected roughly within the %v budget", elapsed, budget)
	}
}

// TestWellKnownFallback ensures /favicon.ico is used when the page declares no
// icon links.
func TestWellKnownFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		_, _ = w.Write(icoBytes(t, 32))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head></head><body></body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	svc := NewFaviconService(dir, testLogger())
	if got := svc.Download(srv.URL); got == "" {
		t.Fatal("expected well-known /favicon.ico to be used")
	}
}
