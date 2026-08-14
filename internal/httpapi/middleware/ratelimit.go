package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rps     float64
	burst   int
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newRateLimiter(rps float64, burst int) *rateLimiter {
	rl := &rateLimiter{buckets: make(map[string]*bucket), rps: rps, burst: burst}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.buckets {
			if now.Sub(b.last) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: float64(rl.burst), last: now}
		rl.buckets[ip] = b
	}
	if rl.rps > 0 {
		elapsed := now.Sub(b.last).Seconds()
		b.tokens += elapsed * rl.rps
		if b.tokens > float64(rl.burst) {
			b.tokens = float64(rl.burst)
		}
	}
	b.last = now

	if rl.rps <= 0 {
		return true
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// clientIP returns the immediate peer address. X-Forwarded-For is deliberately
// ignored: it is client-controllable and would allow bypassing the rate limit.
// If the service is deployed behind a trusted proxy, derive the real client IP
// at the proxy layer instead (e.g. a rewrite that overwrites XFF).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func RateLimit(rps float64, burst int, next http.Handler) http.Handler {
	rl := newRateLimiter(rps, burst)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "rate_limited", "message": "too many requests"},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
