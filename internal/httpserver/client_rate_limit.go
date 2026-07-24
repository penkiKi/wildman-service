package httpserver

import (
	"sync"
	"time"
)

const (
	clientRateLimit  = 120
	clientRateWindow = time.Minute
)

type clientRateEntry struct {
	requests int
	resetAt  time.Time
}

type clientRateLimiter struct {
	mutex       sync.Mutex
	entries     map[string]clientRateEntry
	lastCleanup time.Time
}

func newClientRateLimiter() *clientRateLimiter {
	return &clientRateLimiter{
		entries:     make(map[string]clientRateEntry),
		lastCleanup: time.Now(),
	}
}

func (limiter *clientRateLimiter) Allow(clientID string) (bool, time.Duration) {
	now := time.Now()
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	if now.Sub(limiter.lastCleanup) >= clientRateWindow {
		for entryClientID, entry := range limiter.entries {
			if !now.Before(entry.resetAt) {
				delete(limiter.entries, entryClientID)
			}
		}
		limiter.lastCleanup = now
	}

	entry, exists := limiter.entries[clientID]
	if !exists || !now.Before(entry.resetAt) {
		limiter.entries[clientID] = clientRateEntry{requests: 1, resetAt: now.Add(clientRateWindow)}
		return true, 0
	}
	if entry.requests >= clientRateLimit {
		return false, time.Until(entry.resetAt)
	}
	entry.requests++
	limiter.entries[clientID] = entry
	return true, 0
}
