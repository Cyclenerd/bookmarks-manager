package service

import (
	"io"
	"log/slog"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestFaviconDownloadRejectsNonHTTP(t *testing.T) {
	svc := NewFaviconService(t.TempDir(), testLogger())
	if got := svc.Download("ftp://example.com"); got != "" {
		t.Errorf("expected empty result for non-http scheme, got %q", got)
	}
	if got := svc.Download(""); got != "" {
		t.Errorf("expected empty result for empty url, got %q", got)
	}
}

func TestSanitizeDomain(t *testing.T) {
	if got := sanitizeDomain("example.com:8080"); got != "example.com_8080" {
		t.Errorf("sanitizeDomain = %q", got)
	}
}
