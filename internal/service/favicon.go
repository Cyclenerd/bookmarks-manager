package service

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder

	"golang.org/x/net/html"
)

const (
	maxFaviconBytes = 2 * 1024 * 1024 // 2 MB
	maxFaviconDim   = 512
	thumbSize       = 32
	firefoxUA       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:144.0) Gecko/20100101 Firefox/144.0"
)

// FaviconService downloads, processes and caches website favicons.
type FaviconService struct {
	cacheDir string
	pageCli  *http.Client
	iconCli  *http.Client
	logger   *slog.Logger
}

// NewFaviconService returns a FaviconService caching into cacheDir.
func NewFaviconService(cacheDir string, logger *slog.Logger) *FaviconService {
	return &FaviconService{
		cacheDir: cacheDir,
		pageCli:  &http.Client{Timeout: 10 * time.Second},
		iconCli:  &http.Client{Timeout: 5 * time.Second},
		logger:   logger,
	}
}

// Download attempts to fetch and cache a favicon for rawURL. It returns the
// relative cache path (e.g. "favicons/example.com.png") or "" if none found.
func (s *FaviconService) Download(rawURL string) string {
	if rawURL == "" || !(strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")) {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	domain := parsed.Host
	baseURL := parsed.Scheme + "://" + domain

	// Strategy 1: parse HTML for <link rel="...icon...">.
	if path := s.fromHTML(rawURL, domain); path != "" {
		return path
	}

	// Strategy 2: well-known locations.
	for _, candidate := range []string{
		baseURL + "/favicon.ico",
		baseURL + "/apple-touch-icon.png",
		baseURL + "/favicon.png",
	} {
		if path := s.fetchAndSave(candidate, domain); path != "" {
			return path
		}
	}

	s.logger.Warn("no favicon found", "domain", domain)
	return ""
}

// fromHTML fetches the page, discovers icon links and tries to save one.
func (s *FaviconService) fromHTML(pageURL, domain string) string {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return ""
	}
	s.setHeaders(req)

	resp, err := s.pageCli.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Track redirects so relative URLs resolve correctly.
	finalURL := resp.Request.URL
	domain = finalURL.Host

	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html") {
		return ""
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return ""
	}

	for _, iconURL := range iconLinks(doc, finalURL) {
		if path := s.fetchAndSave(iconURL, domain); path != "" {
			return path
		}
	}
	return ""
}

// iconLinks returns absolute icon URLs discovered in the document.
func iconLinks(doc *html.Node, base *url.URL) []string {
	var links []string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(a.Val)
				case "href":
					href = a.Val
				}
			}
			if strings.Contains(rel, "icon") && href != "" {
				if ref, err := url.Parse(href); err == nil {
					links = append(links, base.ResolveReference(ref).String())
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return links
}

// fetchAndSave downloads iconURL and, if it is an image, saves a thumbnail.
func (s *FaviconService) fetchAndSave(iconURL, domain string) string {
	req, err := http.NewRequest(http.MethodGet, iconURL, nil)
	if err != nil {
		return ""
	}
	s.setHeaders(req)

	resp, err := s.iconCli.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(ct, "image") && !strings.Contains(ct, "octet-stream") {
		return ""
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconBytes+1))
	if err != nil || len(content) > maxFaviconBytes {
		return ""
	}
	return s.save(content, domain)
}

// save decodes, validates, resizes to 32x32 and writes the favicon as PNG.
func (s *FaviconService) save(content []byte, domain string) string {
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return ""
	}
	b := img.Bounds()
	if b.Dx() > maxFaviconDim || b.Dy() > maxFaviconDim {
		return ""
	}

	dst := thumbnail(img)

	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return ""
	}
	filename := sanitizeDomain(domain) + ".png"
	filepath := filepath.Join(s.cacheDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return ""
	}
	defer f.Close()
	if err := png.Encode(f, dst); err != nil {
		return ""
	}
	return "favicons/" + filename
}

// thumbnail scales img to fit within thumbSize x thumbSize, preserving aspect
// ratio using high-quality Catmull-Rom interpolation.
func thumbnail(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= thumbSize && h <= thumbSize {
		return img
	}
	scale := float64(thumbSize) / float64(w)
	if float64(thumbSize)/float64(h) < scale {
		scale = float64(thumbSize) / float64(h)
	}
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

func sanitizeDomain(domain string) string {
	r := strings.NewReplacer(":", "_", "/", "_")
	return r.Replace(domain)
}

func (s *FaviconService) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", firefoxUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8,de-DE;q=0.5,de;q=0.3")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
}
