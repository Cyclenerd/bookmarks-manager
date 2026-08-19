package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestBasicAuth(t *testing.T) {
	h := BasicAuth("admin", "secret")(okHandler())

	// No credentials -> 401.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("missing WWW-Authenticate header")
	}

	// Correct credentials -> 200.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secret")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Wrong password -> 401.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "wrong")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityHeaders(okHandler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, key := range []string{
		"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy",
		"Referrer-Policy", "Strict-Transport-Security",
	} {
		if rec.Header().Get(key) == "" {
			t.Errorf("missing security header %s", key)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	h := rl.Middleware(okHandler())

	do := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if do() != http.StatusOK || do() != http.StatusOK {
		t.Errorf("first two requests should pass")
	}
	if do() != http.StatusTooManyRequests {
		t.Errorf("third request should be rate limited")
	}
}
