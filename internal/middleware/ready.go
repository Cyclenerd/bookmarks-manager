package middleware

import (
	"net/http"
	"sync/atomic"
)

// ReadyGate allows a server to accept TCP connections immediately (so the
// serverless platform marks the instance as started) while deferring real
// request handling until initialisation has finished.
//
// This is important on Cloud Run with min-instances=0: the platform considers
// an instance ready as soon as it listens on $PORT. Binding the socket first
// and swapping in the real handler once initialisation completes lets the
// (boosted) startup CPU overlap with slow work such as opening a
// network-mounted SQLite database, shortening perceived cold-start latency.
type ReadyGate struct {
	handler atomic.Pointer[http.Handler]
}

// NewReadyGate returns a gate that responds 503 until SetHandler is called.
func NewReadyGate() *ReadyGate {
	return &ReadyGate{}
}

// SetHandler installs the real handler, after which all requests are served by
// it. It is safe to call concurrently with ServeHTTP.
func (g *ReadyGate) SetHandler(h http.Handler) {
	g.handler.Store(&h)
}

// ServeHTTP serves the installed handler, or a lightweight 503 while the
// application is still initialising.
func (g *ReadyGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if hp := g.handler.Load(); hp != nil {
		(*hp).ServeHTTP(w, r)
		return
	}
	w.Header().Set("Retry-After", "1")
	http.Error(w, "Starting up", http.StatusServiceUnavailable)
}
