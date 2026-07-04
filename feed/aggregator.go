package feed

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

// Aggregator coordinates fetching data and updating the cache.
type Aggregator struct {
	fetcher nico.VideoFetcher
	cache   *Cache
}

// NewAggregator creates a new Aggregator.
func NewAggregator(fetcher nico.VideoFetcher, cache *Cache) *Aggregator {
	return &Aggregator{
		fetcher: fetcher,
		cache:   cache,
	}
}

// Update fetches videos for all tags in a feed, merges, deduplicates, sorts,
// and updates the cache. If a fetch error occurs, the old cache is preserved.
func (a *Aggregator) Update(ctx context.Context, feedName string, cfg config.FeedConfig) error {
	var allVideos []nico.Video

	for _, tag := range cfg.Tags {
		videos, err := a.fetcher.FetchByTag(ctx, tag)
		if err != nil {
			slog.Error("failed to fetch videos for tag", "tag", tag, "feed", feedName, "error", err)
			return err
		}
		allVideos = append(allVideos, videos...)
	}

	// Deduplicate by video ID
	uniqueVideos := make(map[string]nico.Video)
	for _, v := range allVideos {
		// If duplicate, could keep newest or just one, they represent the same video.
		uniqueVideos[v.ID] = v
	}

	mergedVideos := make([]nico.Video, 0, len(uniqueVideos))
	for _, v := range uniqueVideos {
		mergedVideos = append(mergedVideos, v)
	}

	// Sort by PubDate descending (newest first)
	sort.Slice(mergedVideos, func(i, j int) bool {
		return mergedVideos[i].PubDate.After(mergedVideos[j].PubDate)
	})

	a.cache.Set(feedName, &CachedFeed{
		Videos:      mergedVideos,
		LastUpdated: time.Now(),
	})

	return nil
}
