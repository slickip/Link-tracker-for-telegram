package helpers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultRateLimitCleanupInterval = time.Minute
	defaultRateLimitTTL             = 5 * time.Minute
	defaultRateLimitRequestsPerSec  = 10.0
	defaultRateLimitBurst           = 20
)

type RateLimiterConfig struct {
	Enabled           bool
	RequestsPerSecond float64
	Burst             int
	CleanupInterval   time.Duration
	TTL               time.Duration
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu              sync.Mutex
	limit           rate.Limit
	burst           int
	ttl             time.Duration
	cleanupInterval time.Duration
	lastCleanup     time.Time
	limiters        map[string]*ipLimiter
}

func NewIPRateLimiter(cfg RateLimiterConfig) *IPRateLimiter {
	cfg = NormalizeRateLimiterConfig(cfg)

	return &IPRateLimiter{
		limit:           rate.Limit(cfg.RequestsPerSecond),
		burst:           cfg.Burst,
		ttl:             cfg.TTL,
		cleanupInterval: cfg.CleanupInterval,
		lastCleanup:     time.Now(),
		limiters:        make(map[string]*ipLimiter),
	}
}

func NormalizeRateLimiterConfig(cfg RateLimiterConfig) RateLimiterConfig {
	if cfg.RequestsPerSecond <= 0 {
		cfg.RequestsPerSecond = defaultRateLimitRequestsPerSec
	}

	if cfg.Burst <= 0 {
		cfg.Burst = defaultRateLimitBurst
	}

	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = defaultRateLimitCleanupInterval
	}

	if cfg.TTL <= 0 {
		cfg.TTL = defaultRateLimitTTL
	}

	return cfg
}

func (l *IPRateLimiter) Allow(ip string) bool {
	now := time.Now()

	l.mu.Lock()
	l.cleanupExpiredLocked(now)

	visitor, ok := l.limiters[ip]
	if !ok {
		visitor = &ipLimiter{
			limiter: rate.NewLimiter(l.limit, l.burst),
		}
		l.limiters[ip] = visitor
	}

	visitor.lastSeen = now
	limiter := visitor.limiter

	l.mu.Unlock()

	return limiter.Allow()
}

func (l *IPRateLimiter) cleanupExpiredLocked(now time.Time) {
	if now.Sub(l.lastCleanup) < l.cleanupInterval {
		return
	}

	for ip, visitor := range l.limiters {
		if now.Sub(visitor.lastSeen) > l.ttl {
			delete(l.limiters, ip)
		}
	}

	l.lastCleanup = now
}

func RateLimitMiddleware(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	limiter := NewIPRateLimiter(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)

			if !limiter.Allow(ip) {
				http.Error(
					w,
					http.StatusText(http.StatusTooManyRequests),
					http.StatusTooManyRequests,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ClientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
