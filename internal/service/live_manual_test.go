//go:build livefavicon

package service

import (
	"testing"
)

func TestLiveFavicons(t *testing.T) {
	svc := NewFaviconService(t.TempDir(), testLogger())
	sites := []string{
		"https://github.com",
		"https://news.ycombinator.com",
		"https://www.wikipedia.org",
		"https://stackoverflow.com",
		"https://go.dev",
		"https://www.cloudflare.com",
		"https://www.reddit.com",
		"https://www.nytimes.com",
		"https://developer.mozilla.org",
		"https://www.amazon.com",
	}
	ok := 0
	for _, u := range sites {
		res := svc.Download(u)
		status := "MISS"
		if res != "" {
			status = "ok  "
			ok++
		}
		t.Logf("[%s] %-40s -> %s", status, u, res)
	}
	t.Logf("%d/%d found", ok, len(sites))
}
