package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the application.
//
// Every field can be overridden via an environment variable; where an
// environment variable is absent a sensible default is used.
type Config struct {
	// DatabasePath is the path to the SQLite database file.
	DatabasePath string
	// Debug enables verbose logging.
	Debug bool
	// FaviconCacheDir is the directory where downloaded favicons are stored.
	FaviconCacheDir string
	// AuthUsername is the HTTP Basic Auth username.
	AuthUsername string
	// AuthPassword is the HTTP Basic Auth password.
	AuthPassword string
	// Port is the TCP port the HTTP server listens on.
	Port int
	// MaxContentLength is the maximum accepted request body size in bytes.
	MaxContentLength int64
	// SessionLifetime is retained for parity with the original config.
	SessionLifetime time.Duration
	// RateLimit is the number of requests permitted per RateLimitWindow.
	RateLimit int
	// RateLimitWindow is the sliding window for rate limiting.
	RateLimitWindow time.Duration
	// SecretKey is a random secret used for future signing needs.
	SecretKey string
}

// Load reads configuration from the environment and returns a populated
// Config. It never fails: missing values fall back to defaults.
func Load() *Config {
	return &Config{
		DatabasePath:     env("DATABASE_PATH", "database/bookmarks.db"),
		Debug:            envBool("DEBUG", true),
		FaviconCacheDir:  env("FAVICON_CACHE_DIR", "web/static/favicons"),
		AuthUsername:     env("HTTP_AUTH_USERNAME", "admin"),
		AuthPassword:     env("HTTP_AUTH_PASSWORD", "changeme"),
		Port:             envInt("HTTP_PORT", 8080),
		MaxContentLength: 128 * 1024, // 128 KB
		SessionLifetime:  time.Hour,
		RateLimit:        envInt("RATELIMIT_PER_MINUTE", 100),
		RateLimitWindow:  time.Minute,
		SecretKey:        env("SECRET_KEY", randomHex(32)),
	}
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "insecure-development-secret"
	}
	return hex.EncodeToString(b)
}
