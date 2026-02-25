package middleware

import (
	"net/http"
	"sync"
	"time"
)

type windowEntry struct {
	requests  []time.Time
	mu        sync.Mutex
}

type RateLimiter struct {
	max    int
	window time.Duration
	store  sync.Map
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{max: max, window: window}
}

func (rl *RateLimiter) allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	v, _ := rl.store.LoadOrStore(ip, &windowEntry{})
	entry := v.(*windowEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Prune old requests outside the window
	filtered := entry.requests[:0]
	for _, t := range entry.requests {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	entry.requests = filtered

	if len(entry.requests) >= rl.max {
		return false
	}

	entry.requests = append(entry.requests, now)
	return true
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}
		if !rl.allow(ip) {
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
