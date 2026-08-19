# AGENTS.md

This file provides guidance to AI agents when working with code in this repository.

## System Architecture

This document provides a high-level overview of the `bookmarks` project, a self-hosted bookmark manager written in Go.

## Project Overview

**Bookmarks Manager** is a single-user web application for managing bookmarks with:
- Hierarchical folder organization (unlimited nesting)
- Multi-tag support
- Full-text search with live results
- Keyboard shortcuts for power users
- Mobile-optimized responsive UI
- HTTP Basic Authentication

## Components

### 1. Go Application (`cmd/`, `internal/`, `web/`)

**Architecture Pattern**: Layered / clean architecture

- **`cmd/bookmarks/`**: Program entrypoint, dependency wiring, graceful shutdown,
  and the static-asset overlay filesystem.
- **`internal/config/`**: Loads configuration from environment variables.
- **`internal/database/`**: SQLite connection management and idempotent schema.
- **`internal/models/`**: Domain types (Bookmark, Folder, Tag, ...).
- **`internal/repository/`**: Data access (SQL) for bookmarks, folders, tags.
- **`internal/service/`**: Business logic — favicon download/resize, page
  metadata extraction, Firefox import/export.
- **`internal/middleware/`**: Security headers, HTTP Basic Auth, rate limiting.
- **`internal/handler/`**: HTTP handlers, routing, template rendering, view models.
- **`web/templates/`**: `html/template` files (converted from the Jinja2 originals).
- **`web/static/`**: CSS, fonts, images (Bootstrap 5 dark mode).

### 2. Database Layer
- **Engine**: SQLite via `modernc.org/sqlite` (pure Go, no CGO).
- **Schema**: UUID strings for all primary keys.
- **Tables**: bookmarks, folders, tags, bookmark_tags.
- **Features**: Hierarchical folders (self-referencing), many-to-many tags.
- **Note**: `created_at` doubles as `updated_at` via UPDATE triggers (preserved
  from the original design).

### 3. Frontend Layer
- **Framework**: Bootstrap 5 (Dark Mode), served from embedded assets.
- **Approach**: Server-rendered HTML forms with minimal JavaScript.
- **JavaScript**: Only for live search and keyboard shortcuts (inline in templates).
- **Responsive**: Mobile-first design.

## Directory Structure

```
bookmarks-manager/
├── cmd/
│   └── bookmarks/
│       ├── main.go            # Entrypoint, wiring, graceful shutdown
│       └── static.go          # Overlay filesystem (disk favicons + embedded assets)
├── internal/
│   ├── config/                # Env-var configuration
│   ├── database/              # SQLite connection + schema
│   ├── models/                # Domain types
│   ├── repository/            # Data access (bookmark, folder, tag) + tests
│   ├── service/               # favicon, metadata, firefox + tests
│   ├── middleware/            # auth, security headers, rate limit + tests
│   └── handler/               # HTTP handlers, routing, rendering, view models + tests
├── web/
│   ├── embed.go               # go:embed of templates + static assets
│   ├── templates/             # html/template files
│   └── static/                # css, fonts, img, favicons cache (runtime)
├── Dockerfile                 # Multi-stage, CGO-free Go build on Alpine
├── docker-compose.yml
├── go.mod / go.sum
└── database/                  # SQLite DB location (volume mount)
```

## Technologies Used

*   **Backend:** Go, `net/http` (std lib router with method+path patterns), SQLite
*   **Database Driver:** `modernc.org/sqlite` (pure Go)
*   **Templating:** `html/template`
*   **HTML Parsing:** `golang.org/x/net/html` (favicon links, page titles)
*   **Image Processing:** `image`, `golang.org/x/image/draw` (favicon resize), `golang.org/x/image/webp`
*   **UUIDs:** `github.com/google/uuid`
*   **Frontend:** HTML5, `html/template`, JavaScript (ES6), Bootstrap 5 (Dark Mode)
*   **Authentication:** HTTP Basic Auth (constant-time comparison)

## Go Coding Style

Follow these coding style rules when writing Go code:

*   **Format:** Code must pass `gofmt -l` (no output). Run `gofmt -w ./...`.
*   **Vet:** Code must pass `go vet ./...`.
*   **Modules:** Keep `go.mod`/`go.sum` tidy with `go mod tidy`.
*   **Docs:** Every exported type, function and package has a doc comment.
*   **Errors:** Wrap errors with `%w`; return errors rather than panicking.
*   **Layering:** Handlers call repositories/services; SQL lives only in `repository`.

## Configuration (environment variables)

| Variable | Default | Purpose |
|---|---|---|
| `HTTP_AUTH_USERNAME` | `admin` | Basic Auth username |
| `HTTP_AUTH_PASSWORD` | `changeme` | Basic Auth password |
| `HTTP_PORT` | `8080` | Listen port |
| `DEBUG` | `true` | Verbose (debug) logging |
| `DATABASE_PATH` | `database/bookmarks.db` | SQLite file path (`:memory:` for tests) |
| `FAVICON_CACHE_DIR` | `web/static/favicons` | Favicon cache directory |
| `RATELIMIT_PER_MINUTE` | `100` | Requests per minute per client IP |
| `SECRET_KEY` | random | Reserved for future signing needs |

## HTTP Endpoints (unchanged from the original)

| Method | Path | Purpose |
|---|---|---|
| GET | `/` | Main listing (`folder`, `tag`, `search`, `sort`, `order`, `page`) |
| GET | `/search` | Full-text search page |
| GET | `/api/search` | Live search JSON (min 2 chars, max 10 results) |
| POST | `/api/fetch-metadata` | Fetch page title (JSON body `{url}`) |
| GET | `/bookmark/add` | Add bookmark form |
| GET | `/bookmark/{id}/edit` | Edit bookmark form |
| POST | `/bookmark/save` | Create/update bookmark |
| POST | `/bookmark/{id}/delete` | Delete bookmark |
| POST | `/bookmark/{id}/toggle-pin` | Toggle pinned |
| GET/POST | `/folder/...` | Folder add/edit/save/delete |
| GET/POST | `/tag/...` | Tag add/edit/save/delete |
| GET | `/import` | Import/export page |
| POST | `/import/firefox` | Import Firefox JSON (max 128 KB) |
| GET | `/export/firefox` | Export Firefox JSON |
| GET | `/robots.txt`, `/favicon.ico` | Static (no auth) |

## Testing

*   **Framework:** Standard `testing` package.
*   **Location:** Tests live next to the code they cover (`*_test.go`).
*   **Database:** Tests use an in-memory SQLite database (`:memory:`).
*   **Run all tests:** `go test ./...`
*   **Coverage:** `go test -cover ./...`
*   **Always run tests after making changes.**

Example commands:
```bash
# Build the binary (assets are embedded)
go build -o bookmarks ./cmd/bookmarks

# Run locally
HTTP_AUTH_PASSWORD=changeme ./bookmarks

# Run the full test suite
go test ./...

# Format and vet
gofmt -w ./... && go vet ./...
```

## Code Modification Guidelines

When modifying code:

1. **Preserve Endpoints & Layout**: Keep HTTP routes and rendered HTML compatible.
2. **Preserve Form-Based Approach**: Don't add unnecessary JavaScript.
3. **Use Bootstrap Native**: Don't create custom CSS for layouts.
4. **Maintain Keyboard Shortcuts**: Keep existing shortcuts working.
5. **UUID Consistency**: Use UUIDs for all database primary keys.
6. **Layering**: Put SQL in repositories, business logic in services, HTTP in handlers.
7. **Test Coverage**: Add tests for new features.
8. **Mobile Responsive**: Test on mobile viewports.

## Terraform / Bash Coding Style

The Terraform (`gcp/`) and shell (`tools/`, `gcp/`) style rules are unchanged:

*   **Terraform:** `terraform fmt -recursive -check -diff gcp`, `tflint --chdir gcp`, `tfsec gcp`.
*   **Shell:** `shellcheck tools/*.sh && shellcheck gcp/*.sh`, indent with tabs.
