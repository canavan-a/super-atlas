// Package ratelimit is a fixed-window rate limiter keyed by anything,
// following super-chat's server/api limiter.
package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*window
	now    func() time.Time
}

type window struct {
	start time.Time
	n     int
}

func New(limit int, per time.Duration) *Limiter {
	return &Limiter{limit: limit, window: per, hits: map[string]*window{}, now: time.Now}
}

// Allow records a hit for key and reports whether it's within the limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.hits) > 10000 {
		for k, w := range l.hits {
			if now.Sub(w.start) > l.window {
				delete(l.hits, k)
			}
		}
	}
	w, ok := l.hits[key]
	if !ok || now.Sub(w.start) > l.window {
		l.hits[key] = &window{start: now, n: 1}
		return true
	}
	w.n++
	return w.n <= l.limit
}

// IPKeyer returns the rate-limit key for a request: its remote address, or,
// when trustProxy is set and the connection is from loopback (a local tunnel
// or reverse proxy), the rightmost X-Forwarded-For entry. Only that entry
// can be trusted, since it's the one the proxy appended.
func IPKeyer(trustProxy bool) func(r *http.Request) string {
	return func(r *http.Request) string {
		remote, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			remote = r.RemoteAddr
		}
		if trustProxy && isLoopback(remote) {
			if ip := lastForwardedFor(r.Header.Values("X-Forwarded-For")); ip != "" {
				return ip
			}
		}
		return remote
	}
}

// Middleware rejects a request with 429 once key(r) goes over l.
func Middleware(l *Limiter, key func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow(key(r)) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"slow down"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isLoopback(ip string) bool {
	p := net.ParseIP(ip)
	return p != nil && p.IsLoopback()
}

func lastForwardedFor(values []string) string {
	all := strings.Split(strings.Join(values, ","), ",")
	for i := len(all) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(all[i]); s != "" {
			if p := net.ParseIP(s); p != nil {
				return p.String()
			}
			return ""
		}
	}
	return ""
}
