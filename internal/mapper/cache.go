package mapper

import (
	"sync"
	"time"
)

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	maxSize int
}

func NewCache(ttl time.Duration, maxSize int) *Cache {
	c := &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
	go c.cleanup()
	return c
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *Cache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for k, v := range c.entries {
		if first || v.expiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.expiresAt
			first = false
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *Cache) cleanup() {
	for {
		time.Sleep(30 * time.Second)
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
