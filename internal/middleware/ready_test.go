package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyGate(t *testing.T) {
	gate := NewReadyGate()

	// Before a handler is set, the gate reports 503 with Retry-After.
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before ready, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Errorf("expected Retry-After header while starting up")
	}

	// After the handler is installed, requests are served by it.
	gate.SetHandler(okHandler())
	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after ready, got %d", rec.Code)
	}
}
