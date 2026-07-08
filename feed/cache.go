package feed

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

// CachedFeed represents the in-memory cache data for a single feed.
type CachedFeed struct {
	Videos      []nico.Video `json:"videos"`
	RSSXML      []byte       `json:"rss_xml"`
	LastUpdated time.Time    `json:"last_updated"`
	ETag        string       `json:"etag"`
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

// DumpToFile serializes the cache to a JSON file.
func (c *Cache) DumpToFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c.feeds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// LoadFromFile deserializes the cache from a JSON file.
func (c *Cache) LoadFromFile(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	if err := json.Unmarshal(data, &c.feeds); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	return nil
}

// CleanExpired removes videos older than the specified retention period from a feed.
func (c *Cache) CleanExpired(feedName string, retentionDays int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	feed, ok := c.feeds[feedName]
	if !ok {
		return
	}

	cutoffTime := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	filtered := make([]nico.Video, 0, len(feed.Videos))
	for _, v := range feed.Videos {
		if v.PubDate.After(cutoffTime) {
			filtered = append(filtered, v)
		}
	}

	feed.Videos = filtered
}
