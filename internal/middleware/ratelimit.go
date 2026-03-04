package middleware

import (
	"net/http"
	"strings"
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
	rl := &RateLimiter{max: max, window: window}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rl.window)
		rl.store.Range(func(key, value any) bool {
			entry := value.(*windowEntry)
			entry.mu.Lock()
			stale := len(entry.requests) == 0 || entry.requests[len(entry.requests)-1].Before(cutoff)
			entry.mu.Unlock()
			if stale {
				rl.store.Delete(key)
			}
			return true
		})
	}
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
			// Take only the first (client) IP from a potentially spoofed chain
			ip = strings.TrimSpace(strings.SplitN(forwarded, ",", 2)[0])
		}
		if !rl.allow(ip) {
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
