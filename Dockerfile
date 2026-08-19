# syntax=docker/dockerfile:1

# --- Build Stage ---
# Compiles a static Go binary. Templates and static assets are embedded into
# the binary via go:embed, so the final image only needs the binary itself.
ARG GO_VERSION=1.26
FROM docker.io/library/golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

# Cache dependencies separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

# Copy the remainder of the source (including web/ for go:embed).
COPY . .

# CGO is disabled: modernc.org/sqlite is pure Go, so no C toolchain is needed.
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /bin/bookmarks ./cmd/bookmarks

# --- Production Stage ---
FROM docker.io/library/alpine:3.20

# CA certificates are required for outbound HTTPS (favicon/metadata fetching).
RUN apk add --no-cache ca-certificates && \
    addgroup -S appuser && adduser -S -G appuser appuser

ENV HTTP_PORT=8080 \
    DEBUG=false \
    DATABASE_PATH=/var/lib/bookmarks/bookmarks.db \
    FAVICON_CACHE_DIR=/var/lib/bookmarks/favicons

WORKDIR /app
COPY --from=builder /bin/bookmarks /app/bookmarks

# Data directory for the SQLite database and favicon cache (mount a volume).
RUN mkdir -p /var/lib/bookmarks/favicons && chown -R appuser:appuser /var/lib/bookmarks /app

USER appuser
EXPOSE 8080

ENTRYPOINT ["/app/bookmarks"]
