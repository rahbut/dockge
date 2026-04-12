// Package ratelimit provides per-IP token-bucket rate limiters, matching the
// behaviour of the original Node.js limiter-es6-compat implementation:
//   - Login:  20 requests / minute
//   - API:    60 requests / minute
//
// Stale per-IP entries are cleaned up by a background goroutine every 10 minutes.
package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter is a map of per-IP rate limiters.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit
	b        int
}

// NewIPRateLimiter creates a new per-IP limiter with the given rate (events/s)
// and burst capacity.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	l := &IPRateLimiter{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		b:        b,
	}
	go l.cleanup()
	return l
}

// Allow reports whether the given IP address is allowed to proceed.
func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	entry, ok := l.limiters[ip]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(l.r, l.b)}
		l.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	lim := entry.limiter
	l.mu.Unlock()
	return lim.Allow()
}

// Wait blocks until the given IP is allowed to proceed (or ctx is cancelled).
func (l *IPRateLimiter) Wait(ctx context.Context, ip string) error {
	l.mu.Lock()
	entry, ok := l.limiters[ip]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(l.r, l.b)}
		l.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	lim := entry.limiter
	l.mu.Unlock()
	return lim.Wait(ctx)
}

// cleanup removes stale entries every 10 minutes.
func (l *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, entry := range l.limiters {
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(l.limiters, ip)
			}
		}
		l.mu.Unlock()
	}
}

// Pre-configured limiters matching the original Node.js configuration.
var (
	// LoginLimiter allows 20 login attempts per minute per IP (burst 20).
	LoginLimiter = NewIPRateLimiter(rate.Every(time.Minute/20), 20)

	// APILimiter allows 60 API calls per minute per IP (burst 60).
	APILimiter = NewIPRateLimiter(rate.Every(time.Minute/60), 60)
)
