package service

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	_ "github.com/biessek/golang-ico" // register ICO decoder (favicon.ico)
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder

	"golang.org/x/net/html"
)

const (
	maxFaviconBytes = 2 * 1024 * 1024 // 2 MB
	thumbSize       = 32
	firefoxUA       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:144.0) Gecko/20100101 Firefox/144.0"

	// defaultTotalBudget bounds the entire favicon lookup (page fetch + all
	// well-known probes + fallback). This is the hard ceiling on how long a
	// bookmark save can block waiting on favicon discovery.
	defaultTotalBudget = 8 * time.Second
)

// FaviconService downloads, processes and caches website favicons.
type FaviconService struct {
	cacheDir    string
	client      *http.Client
	totalBudget time.Duration
	logger      *slog.Logger
}

// NewFaviconService returns a FaviconService caching into cacheDir with the
// default total time budget.
func NewFaviconService(cacheDir string, logger *slog.Logger) *FaviconService {
	return NewFaviconServiceWithBudget(cacheDir, defaultTotalBudget, logger)
}

// NewFaviconServiceWithBudget returns a FaviconService that spends at most
// totalBudget on a single Download call across all network attempts.
//
// A single shared client is used; the overall deadline is enforced via a
// context so that a site with no favicon (or a slow/unreachable one) fails fast
// instead of accumulating a separate timeout for every candidate URL.
func NewFaviconServiceWithBudget(cacheDir string, totalBudget time.Duration, logger *slog.Logger) *FaviconService {
	if totalBudget <= 0 {
		totalBudget = defaultTotalBudget
	}
	return &FaviconService{
		cacheDir: cacheDir,
		// The per-request timeout is a safety net; the real limit is the
		// per-Download context deadline, shared across all attempts.
		client:      &http.Client{Timeout: totalBudget},
		totalBudget: totalBudget,
		logger:      logger,
	}
}

// iconCandidate is a discovered icon URL with a priority score (higher is
// preferred) used to try the most promising icons first.
type iconCandidate struct {
	url   string
	score int
}

// Download attempts to fetch and cache a favicon for rawURL using a background
// context bounded by the service's total time budget.
func (s *FaviconService) Download(rawURL string) string {
	return s.DownloadContext(context.Background(), rawURL)
}

// DownloadContext attempts to fetch and cache a favicon for rawURL. It returns
// the relative cache path (e.g. "favicons/example.com.png") or "" if none is
// found within the total time budget.
//
// It tries, in order (all sharing one deadline):
//  1. Icons declared in the page's HTML (<link rel="...icon...">), best first.
//  2. Well-known locations (/favicon.ico, /apple-touch-icon.png, ...).
//  3. Google's public favicon service as a last resort.
//
// The whole operation is bounded by the total budget so that a bookmark with no
// favicon (or a slow host) does not block the save for tens of seconds.
func (s *FaviconService) DownloadContext(ctx context.Context, rawURL string) string {
	if rawURL == "" || !(strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")) {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	domain := parsed.Host

	// One deadline for the entire lookup, shared by every network attempt.
	ctx, cancel := context.WithTimeout(ctx, s.totalBudget)
	defer cancel()

	// Strategy 1: parse the HTML for declared icons. This also returns the
	// final (possibly redirected) URL so well-known locations resolve against
	// the correct host.
	candidates, finalURL := s.iconsFromHTML(ctx, rawURL)
	if finalURL != nil {
		domain = finalURL.Host
	} else {
		finalURL = parsed
	}
	for _, c := range candidates {
		if ctx.Err() != nil {
			break
		}
		if path := s.fetchAndSave(ctx, c.url, domain); path != "" {
			return path
		}
	}

	// Strategy 2: well-known locations relative to the final base URL.
	base := finalURL.Scheme + "://" + finalURL.Host
	for _, candidate := range []string{
		base + "/favicon.ico",
		base + "/apple-touch-icon.png",
		base + "/apple-touch-icon-precomposed.png",
		base + "/favicon.png",
		base + "/favicon.svg",
	} {
		if ctx.Err() != nil {
			break
		}
		if path := s.fetchAndSave(ctx, candidate, domain); path != "" {
			return path
		}
	}

	// Strategy 3: Google's favicon service. Reliable for sites that block bots
	// or render their <head> with JavaScript.
	if ctx.Err() == nil {
		if path := s.fetchAndSave(ctx, googleFaviconURL(domain), domain); path != "" {
			s.logger.Debug("favicon via google fallback", "domain", domain)
			return path
		}
	}

	s.logger.Warn("no favicon found", "domain", domain, "timed_out", ctx.Err() != nil)
	return ""
}

// iconsFromHTML fetches the page and returns declared icon URLs ordered by
// preference, along with the final URL after any redirects.
func (s *FaviconService) iconsFromHTML(ctx context.Context, pageURL string) ([]iconCandidate, *url.URL) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, nil
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL
	if resp.StatusCode != http.StatusOK {
		return nil, finalURL
	}

	// Accept HTML and XHTML; some servers omit or misreport the content type,
	// so an empty content type is tolerated and left to the parser.
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "html") && !strings.Contains(ct, "xml") {
		return nil, finalURL
	}

	// The body is transparently decompressed by the HTTP transport because we
	// deliberately do NOT set Accept-Encoding ourselves.
	doc, err := html.Parse(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, finalURL
	}
	return iconLinks(doc, finalURL), finalURL
}

// iconLinks extracts and scores icon <link> elements from the document,
// returning absolute URLs ordered best-first.
func iconLinks(doc *html.Node, base *url.URL) []iconCandidate {
	var candidates []iconCandidate
	seen := map[string]bool{}

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href, sizes, typ string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(a.Val)
				case "href":
					href = strings.TrimSpace(a.Val)
				case "sizes":
					sizes = strings.ToLower(a.Val)
				case "type":
					typ = strings.ToLower(a.Val)
				}
			}
			if href != "" && isIconRel(rel) {
				if ref, err := url.Parse(href); err == nil {
					abs := base.ResolveReference(ref).String()
					if !seen[abs] {
						seen[abs] = true
						candidates = append(candidates, iconCandidate{
							url:   abs,
							score: scoreIcon(rel, sizes, typ),
						})
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Stable sort by descending score so document order is preserved for ties.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	return candidates
}

// isIconRel reports whether a rel attribute denotes a usable icon.
func isIconRel(rel string) bool {
	for _, tok := range strings.Fields(rel) {
		switch tok {
		case "icon", "shortcut", "apple-touch-icon", "apple-touch-icon-precomposed", "fluid-icon", "mask-icon":
			return true
		}
	}
	return false
}

// scoreIcon ranks an icon so that crisp, appropriately-sized raster icons are
// preferred. Larger declared sizes win; apple-touch-icons (usually high
// resolution PNGs) are favoured; SVGs are preferred as they scale cleanly.
func scoreIcon(rel, sizes, typ string) int {
	score := 0
	if strings.Contains(rel, "apple-touch-icon") {
		score += 40
	}
	if strings.Contains(typ, "svg") || strings.HasSuffix(sizes, ".svg") {
		score += 30
	}
	// Parse the largest declared dimension, e.g. "32x32" or "16x16 32x32".
	best := 0
	for _, tok := range strings.Fields(sizes) {
		if tok == "any" {
			best = 64 // treat scalable "any" as a decent size
			continue
		}
		if x := strings.IndexByte(tok, 'x'); x > 0 {
			if n, err := strconv.Atoi(tok[:x]); err == nil && n > best {
				best = n
			}
		}
	}
	switch {
	case best >= 180:
		score += 25
	case best >= 96:
		score += 20
	case best >= 48:
		score += 15
	case best >= 32:
		score += 12
	case best >= 16:
		score += 5
	}
	return score
}

// googleFaviconURL builds a request to Google's public favicon service.
func googleFaviconURL(domain string) string {
	return "https://www.google.com/s2/favicons?sz=64&domain=" + url.QueryEscape(domain)
}

// fetchAndSave downloads iconURL and, if the bytes decode as an image, saves a
// normalised 32x32 PNG thumbnail.
func (s *FaviconService) fetchAndSave(ctx context.Context, iconURL, domain string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL, nil)
	if err != nil {
		return ""
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Do NOT trust Content-Type: many servers send text/plain, application/ico,
	// an empty type, or text/html error pages for .ico requests. Read the bytes
	// and let the decoder decide whether it is a real image.
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconBytes+1))
	if err != nil || len(content) == 0 || len(content) > maxFaviconBytes {
		return ""
	}
	return s.save(content, domain)
}

// save decodes any supported image (PNG, ICO, JPEG, GIF, WebP, SVG-as-raster is
// not supported), downscales it to fit 32x32, and writes it as PNG.
func (s *FaviconService) save(content []byte, domain string) string {
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return ""
	}
	if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		return ""
	}

	dst := thumbnail(img)

	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return ""
	}
	filename := sanitizeDomain(domain) + ".png"
	fpath := filepath.Join(s.cacheDir, filename)

	f, err := os.Create(fpath)
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
// ratio using high-quality Catmull-Rom interpolation. Icons larger than the
// target are downscaled (previously such icons were rejected outright).
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

// setHeaders sets browser-like request headers.
//
// Note: Accept-Encoding is intentionally NOT set. Go's HTTP transport only
// transparently decompresses gzip responses when the caller leaves
// Accept-Encoding unset; setting it manually would hand back compressed bytes
// that the HTML/image parsers cannot read — a common cause of "no favicon
// found" for sites that gzip their responses.
func (s *FaviconService) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", firefoxUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8,de-DE;q=0.5,de;q=0.3")
	req.Header.Set("DNT", "1")
}
