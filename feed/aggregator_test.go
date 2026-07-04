package feed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

type mockFetcher struct {
	videos map[string][]nico.Video
	err    error
}

func (m *mockFetcher) FetchByTag(ctx context.Context, tag string) ([]nico.Video, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.videos[tag], nil
}

func TestAggregator_Update_MergeSortDeduplicate(t *testing.T) {
	now := time.Now()
	v1 := nico.Video{ID: "sm1", Title: "Video 1", PubDate: now.Add(-10 * time.Minute)}
	v2 := nico.Video{ID: "sm2", Title: "Video 2", PubDate: now.Add(-5 * time.Minute)}
	v3 := nico.Video{ID: "sm3", Title: "Video 3", PubDate: now.Add(-1 * time.Minute)}

	fetcher := &mockFetcher{
		videos: map[string][]nico.Video{
			"tag1": {v2, v1}, // v2 is newer than v1
			"tag2": {v3, v2}, // v2 is a duplicate, v3 is newest
		},
	}

	cache := NewCache()
	agg := NewAggregator(fetcher, cache)

	feedCfg := config.FeedConfig{
		Tags: []string{"tag1", "tag2"},
	}

	err := agg.Update(context.Background(), "test_feed", feedCfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := cache.Get("test_feed")
	if !ok {
		t.Fatal("expected feed to be cached")
	}

	if len(got.Videos) != 3 {
		t.Fatalf("expected 3 videos, got %d", len(got.Videos))
	}

	// Should be sorted by PubDate desc: v3, v2, v1
	if got.Videos[0].ID != "sm3" || got.Videos[1].ID != "sm2" || got.Videos[2].ID != "sm1" {
		t.Errorf("unexpected order or videos: %+v", got.Videos)
	}
}

func TestAggregator_Update_ErrorHandling(t *testing.T) {
	// 既存のキャッシュがある状態で、タグ取得エラーが発生した場合に
	// エラーを返しつつ、古いキャッシュが維持されることを確認する。
	now := time.Now()
	oldVideo := nico.Video{ID: "sm99", Title: "Old Video", PubDate: now}
	
	fetcher := &mockFetcher{
		err: errors.New("network error"),
	}

	cache := NewCache()
	cache.Set("test_feed", &CachedFeed{
		Videos:      []nico.Video{oldVideo},
		LastUpdated: now,
	})

	agg := NewAggregator(fetcher, cache)
	feedCfg := config.FeedConfig{
		Tags: []string{"tag1"},
	}

	err := agg.Update(context.Background(), "test_feed", feedCfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	got, ok := cache.Get("test_feed")
	if !ok {
		t.Fatal("expected cache to still exist")
	}

	if len(got.Videos) != 1 || got.Videos[0].ID != "sm99" {
		t.Error("expected old cache to be preserved")
	}
}
