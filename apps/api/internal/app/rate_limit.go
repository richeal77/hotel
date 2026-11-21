package app

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateEntry struct {
	count int
	reset time.Time
}
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
	now     func() time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{entries: make(map[string]rateEntry), now: time.Now}
}

func (l *rateLimiter) allow(key string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	entry := l.entries[key]
	if !now.Before(entry.reset) {
		entry = rateEntry{reset: now.Add(window)}
	}
	if entry.count >= limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	if len(l.entries) > 10000 {
		for k, v := range l.entries {
			if !now.Before(v.reset) {
				delete(l.entries, k)
			}
		}
	}
	return true
}

func limited(l *rateLimiter, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.allow(c.ClientIP()+"|"+c.FullPath(), limit, window) {
			respond(c, nil, &AppError{Code: CodeTooMany, HTTPStatus: 429, Message: "请求过于频繁"})
			c.Abort()
		}
	}
}
