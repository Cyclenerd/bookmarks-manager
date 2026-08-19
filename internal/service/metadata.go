// Package service contains business logic that is not pure data access:
// favicon downloading, page metadata extraction and Firefox import/export.
package service

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// MetadataService fetches page titles from remote URLs.
type MetadataService struct {
	client *http.Client
}

// NewMetadataService returns a MetadataService with a 10s timeout client.
func NewMetadataService() *MetadataService {
	return &MetadataService{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Metadata is the result of a page metadata fetch.
type Metadata struct {
	Title   string `json:"title"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

const chromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// FetchTitle retrieves the best available title for url, preferring
// og:title, then twitter:title, then the <title> element.
func (s *MetadataService) FetchTitle(url string) Metadata {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Metadata{Success: false, Error: err.Error()}
	}
	req.Header.Set("User-Agent", chromeUA)

	resp, err := s.client.Do(req)
	if err != nil {
		return Metadata{Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return Metadata{Success: false, Error: resp.Status}
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return Metadata{Success: false, Error: err.Error()}
	}

	title := extractTitle(doc)
	return Metadata{Title: title, Success: true}
}

// extractTitle walks the parsed document collecting candidate titles.
func extractTitle(doc *html.Node) string {
	var ogTitle, twitterTitle, titleTag string

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "meta":
				var property, name, content string
				for _, a := range n.Attr {
					switch strings.ToLower(a.Key) {
					case "property":
						property = a.Val
					case "name":
						name = a.Val
					case "content":
						content = a.Val
					}
				}
				if property == "og:title" && content != "" && ogTitle == "" {
					ogTitle = content
				}
				if name == "twitter:title" && content != "" && twitterTitle == "" {
					twitterTitle = content
				}
			case "title":
				if titleTag == "" && n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					titleTag = n.FirstChild.Data
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	for _, t := range []string{ogTitle, twitterTitle, titleTag} {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
