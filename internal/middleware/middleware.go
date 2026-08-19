// Package middleware provides HTTP middleware: security headers, HTTP Basic
// authentication and a simple in-memory rate limiter.
package middleware

import (
	"crypto/subtle"
	"net"
	"net/http"
	"sync"
	"time"
)

// Middleware is a standard http.Handler decorator.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to h in order, so the first listed runs outermost.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// SecurityHeaders sets the same hardening headers as the original app on every
// response.
func SecurityHeaders(next http.Handler) http.Handler {
	const csp = "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"font-src 'self'; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
}

// BasicAuth returns middleware enforcing HTTP Basic Authentication using a
// constant-time comparison of both username and password.
func BasicAuth(username, password string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || !credentialsMatch(user, pass, username, password) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Login Required"`)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func credentialsMatch(gotUser, gotPass, wantUser, wantPass string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(gotUser), []byte(wantUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(gotPass), []byte(wantPass)) == 1
	return userOK && passOK
}

// RateLimiter is a fixed-window per-client-IP rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*window
	limit    int
	interval time.Duration
}

type window struct {
	count int
	reset time.Time
}

// NewRateLimiter creates a limiter allowing limit requests per interval, per
// remote address. A background goroutine periodically prunes stale entries.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*window),
		limit:    limit,
		interval: interval,
	}
	go rl.cleanup()
	return rl
}

// Middleware enforces the rate limit, returning 429 when exceeded.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, ok := rl.visitors[ip]
	if !ok || now.After(v.reset) {
		rl.visitors[ip] = &window{count: 1, reset: now.Add(rl.interval)}
		return true
	}
	if v.count >= rl.limit {
		return false
	}
	v.count++
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if now.After(v.reset) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
