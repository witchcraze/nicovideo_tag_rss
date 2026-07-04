package feed

import (
	"sync"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

// CachedFeed represents the in-memory cache data for a single feed.
type CachedFeed struct {
	Videos      []nico.Video
	RSSXML      []byte
	LastUpdated time.Time
	ETag        string
}

// Cache is a thread-safe in-memory cache for feeds.
type Cache struct {
	mu    sync.RWMutex
	feeds map[string]*CachedFeed
}

// NewCache creates a new Cache instance.
func NewCache() *Cache {
	return &Cache{
		feeds: make(map[string]*CachedFeed),
	}
}

// Get retrieves a cached feed by its name.
func (c *Cache) Get(feedName string) (*CachedFeed, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	feed, ok := c.feeds[feedName]
	return feed, ok
}

// Set stores a feed in the cache.
func (c *Cache) Set(feedName string, feed *CachedFeed) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.feeds[feedName] = feed
}
