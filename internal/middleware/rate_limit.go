package middleware

import (
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/internal/httpx"
)

type rateLimiter struct {
	mu          sync.Mutex
	cfg         config.RateLimitConfig
	buckets     map[string]*rateLimitBucket
	lastCleanup time.Time
}

type rateLimitBucket struct {
	windowStart time.Time
	count       int
	lastSeen    time.Time
}

func RateLimit(cfg config.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled || cfg.Requests <= 0 || cfg.Window <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limiter := &rateLimiter{
		cfg:     cfg,
		buckets: make(map[string]*rateLimitBucket),
	}

	return limiter.handle
}

func (l *rateLimiter) handle(c *gin.Context) {
	now := time.Now()
	key := rateLimitKey(c)

	allowed, remaining, resetAt := l.allow(key, now)
	c.Header("X-RateLimit-Limit", strconv.Itoa(l.cfg.Requests))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

	if !allowed {
		c.Header("Retry-After", strconv.FormatInt(secondsUntil(resetAt, now), 10))
		httpx.Error(c, apperr.TooManyRequests())
		return
	}

	c.Next()
}

func (l *rateLimiter) allow(key string, now time.Time) (bool, int, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now)

	bucket := l.buckets[key]
	if bucket == nil || !now.Before(bucket.windowStart.Add(l.cfg.Window)) {
		bucket = &rateLimitBucket{
			windowStart: now,
		}
		l.buckets[key] = bucket
	}

	resetAt := bucket.windowStart.Add(l.cfg.Window)
	bucket.lastSeen = now

	if bucket.count >= l.cfg.Requests {
		return false, 0, resetAt
	}

	bucket.count++
	remaining := l.cfg.Requests - bucket.count
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, resetAt
}

func (l *rateLimiter) cleanup(now time.Time) {
	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < l.cfg.Window {
		return
	}
	l.lastCleanup = now

	cutoff := now.Add(-2 * l.cfg.Window)
	for key, bucket := range l.buckets {
		if bucket.lastSeen.Before(cutoff) {
			delete(l.buckets, key)
		}
	}
}

func rateLimitKey(c *gin.Context) string {
	if ip := c.ClientIP(); ip != "" {
		return ip
	}

	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return c.Request.RemoteAddr
}

func secondsUntil(target time.Time, now time.Time) int64 {
	duration := target.Sub(now)
	if duration <= 0 {
		return 0
	}
	return int64((duration + time.Second - 1) / time.Second)
}
