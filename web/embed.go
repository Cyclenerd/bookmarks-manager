// Package web embeds the HTML templates and static assets so the application
// ships as a single self-contained binary.
package web

import "embed"

// Files contains the templates and static assets (CSS, fonts, images).
// The runtime favicon cache is intentionally excluded and served from disk.
//
//go:embed templates/*.html
//go:embed static/css/*.css static/css/fonts/* static/img/* static/robots.txt
var Files embed.FS
