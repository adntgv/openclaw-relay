package server

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements rate limiting with two separate limiters:
// - Per-IP for WebSocket connections
// - Per-token for API commands
type RateLimiter struct {
	ipMu        sync.Mutex
	ipLimiters  map[string]*rate.Limiter
	
	tokenMu     sync.Mutex
	tokenLimiters map[string]*rate.Limiter
	
	ipRate      rate.Limit
	ipBurst     int
	tokenRate   rate.Limit
	tokenBurst  int
}

// NewRateLimiter creates a rate limiter with default settings
// IP: 10 req/s burst 30, Token: 5 commands/s burst 10
func NewRateLimiter(ipRate int, ipInterval time.Duration, ipBurst int) *RateLimiter {
	return &RateLimiter{
		ipLimiters:    make(map[string]*rate.Limiter),
		tokenLimiters: make(map[string]*rate.Limiter),
		ipRate:        rate.Every(ipInterval),
		ipBurst:       ipBurst,
		tokenRate:     rate.Every(200 * time.Millisecond), // 5 req/s
		tokenBurst:    10,
	}
}

// getIPLimiter returns or creates a limiter for an IP address
func (rl *RateLimiter) getIPLimiter(ip string) *rate.Limiter {
	rl.ipMu.Lock()
	defer rl.ipMu.Unlock()

	limiter, exists := rl.ipLimiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.ipRate, rl.ipBurst)
		rl.ipLimiters[ip] = limiter
	}

	return limiter
}

// getTokenLimiter returns or creates a limiter for a token
func (rl *RateLimiter) getTokenLimiter(token string) *rate.Limiter {
	rl.tokenMu.Lock()
	defer rl.tokenMu.Unlock()

	limiter, exists := rl.tokenLimiters[token]
	if !exists {
		limiter = rate.NewLimiter(rl.tokenRate, rl.tokenBurst)
		rl.tokenLimiters[token] = limiter
	}

	return limiter
}

// AllowIP checks if a request from an IP is allowed
func (rl *RateLimiter) AllowIP(ip string) bool {
	limiter := rl.getIPLimiter(ip)
	return limiter.Allow()
}

// AllowToken checks if a command from a token/claw_id is allowed
func (rl *RateLimiter) AllowToken(token string) bool {
	limiter := rl.getTokenLimiter(token)
	return limiter.Allow()
}

// Cleanup removes stale limiters
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.ipMu.Lock()
	defer rl.ipMu.Unlock()
	
	// For simplicity, clear all limiters periodically
	// In production, track last access time per limiter
	if len(rl.ipLimiters) > 1000 {
		rl.ipLimiters = make(map[string]*rate.Limiter)
	}
	
	rl.tokenMu.Lock()
	defer rl.tokenMu.Unlock()
	
	if len(rl.tokenLimiters) > 1000 {
		rl.tokenLimiters = make(map[string]*rate.Limiter)
	}
}

// Middleware creates an HTTP middleware that rate-limits by IP
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !rl.AllowIP(ip) {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
