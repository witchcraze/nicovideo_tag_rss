package feed

import (
	"testing"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

func TestCache(t *testing.T) {
	cache := NewCache()

	// Should be empty initially
	if _, ok := cache.Get("test_feed"); ok {
		t.Error("expected cache to be empty")
	}

	// Set and Get
	now := time.Now()
	cf := &CachedFeed{
		Videos:      []nico.Video{{ID: "sm123"}},
		RSSXML:      []byte("<rss></rss>"),
		LastUpdated: now,
		ETag:        "W/\"123\"",
	}

	cache.Set("test_feed", cf)

	got, ok := cache.Get("test_feed")
	if !ok {
		t.Fatal("expected cache to have test_feed")
	}

	if got.ETag != "W/\"123\"" {
		t.Errorf("expected ETag W/\"123\", got %s", got.ETag)
	}
	if len(got.Videos) != 1 || got.Videos[0].ID != "sm123" {
		t.Error("unexpected videos content")
	}
}
