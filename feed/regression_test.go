package feed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

// TestAggregator_Update_AllSortsAndTagsCombined は、複数 sorts × 複数 tags の
// 全組み合わせで FetchByTag が呼ばれることを回帰テストとして固定する。
// （ループ順序：外側 sorts, 内側 tags）
func TestAggregator_Update_AllSortsAndTagsCombined(t *testing.T) {
	now := time.Now()
	fetcher := &mockFetcher{
		videos: map[string][]nico.Video{
			"tagA": {{ID: "smA", PubDate: now}},
			"tagB": {{ID: "smB", PubDate: now.Add(-time.Minute)}},
		},
	}

	cache := NewCache()
	agg := NewAggregator(fetcher, cache)

	feedCfg := config.FeedConfig{
		Tags: []string{"tagA", "tagB"},
		Sorts: []config.SortConfig{
			{ID: "latest", Sort: "registeredAt", Title: "最新"},
			{ID: "popular", Sort: "viewCount", Title: "人気"},
		},
	}

	if err := agg.Update(context.Background(), "test_feed", feedCfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// tagA は registeredAt と viewCount の両方で呼ばれる
	if len(fetcher.called["tagA"]) != 2 {
		t.Errorf("regression: tagA should be fetched 2 times (once per sort), got %d", len(fetcher.called["tagA"]))
	}
	// tagB も同様
	if len(fetcher.called["tagB"]) != 2 {
		t.Errorf("regression: tagB should be fetched 2 times (once per sort), got %d", len(fetcher.called["tagB"]))
	}

	got, ok := cache.Get("test_feed")
	if !ok {
		t.Fatal("expected feed to be cached")
	}
	// smA と smB の2件（重複排除で各1件）
	if len(got.Videos) != 2 {
		t.Errorf("regression: expected 2 unique videos, got %d", len(got.Videos))
	}
}

// TestAggregator_Update_DeduplicateAcrossSorts は、同じ動画 ID が複数の sort から
// 返ってきた場合でも、マージ後は1件のみであることを回帰テストとして固定する。
func TestAggregator_Update_DeduplicateAcrossSorts(t *testing.T) {
	now := time.Now()
	sameVideo := nico.Video{ID: "sm_dup", Title: "Duplicate Video", PubDate: now}

	fetcher := &mockFetcher{
		videos: map[string][]nico.Video{
			"tag1": {sameVideo},
		},
	}

	cache := NewCache()
	agg := NewAggregator(fetcher, cache)

	feedCfg := config.FeedConfig{
		Tags: []string{"tag1"},
		Sorts: []config.SortConfig{
			{ID: "latest", Sort: "registeredAt", Title: "最新"},
			{ID: "popular", Sort: "viewCount", Title: "人気"},
		},
	}

	if err := agg.Update(context.Background(), "test_feed", feedCfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := cache.Get("test_feed")
	if !ok {
		t.Fatal("expected feed to be cached")
	}
	if len(got.Videos) != 1 {
		t.Errorf("regression: same video ID from multiple sorts should be deduplicated to 1, got %d", len(got.Videos))
	}
	if got.Videos[0].ID != "sm_dup" {
		t.Errorf("regression: expected video ID 'sm_dup', got '%s'", got.Videos[0].ID)
	}
}

// TestCleanExpired_ZeroRetentionDays は、retentionDays=0 を指定した場合に
// 全動画（今日投稿のものも含め）が削除されることを回帰テストとして固定する。
// CleanExpired の仕様: cutoff = now - retentionDays*24h
// retentionDays=0 => cutoff ≈ now => 全ての動画が「古い」とみなされる
func TestCleanExpired_ZeroRetentionDays(t *testing.T) {
	cache := NewCache()
	now := time.Now()

	cf := &CachedFeed{
		Videos: []nico.Video{
			{ID: "sm1", PubDate: now},                          // 今日
			{ID: "sm2", PubDate: now.Add(-24 * time.Hour)},     // 1日前
			{ID: "sm3", PubDate: now.Add(-7 * 24 * time.Hour)}, // 7日前
		},
	}
	cache.Set("feed1", cf)

	cache.CleanExpired("feed1", 0)

	got, ok := cache.Get("feed1")
	if !ok {
		t.Fatal("expected feed to exist after CleanExpired")
	}
	// retentionDays=0 では cutoff = now となるので、
	// PubDate が After(now) の動画はなく、全て削除されるはず
	if len(got.Videos) != 0 {
		t.Errorf("regression: with retentionDays=0, all videos should be removed, got %d videos", len(got.Videos))
	}
}

// TestAggregator_Update_PreservesOldCacheOnError は、フェッチエラー発生時に
// 既存のキャッシュが書き換えられないことを回帰テストとして固定する。
func TestAggregator_Update_PreservesOldCacheOnFetchError(t *testing.T) {
	now := time.Now()
	oldVideos := []nico.Video{
		{ID: "sm_old1", PubDate: now.Add(-time.Hour)},
		{ID: "sm_old2", PubDate: now.Add(-2 * time.Hour)},
	}

	fetcher := &mockFetcher{
		err: fmt.Errorf("simulated error"),
	}

	cache := NewCache()
	cache.Set("test_feed", &CachedFeed{
		Videos:      oldVideos,
		LastUpdated: now,
		ETag:        `W/"old-etag"`,
	})

	agg := NewAggregator(fetcher, cache)
	feedCfg := config.FeedConfig{
		Tags:  []string{"tag1"},
		Sorts: []config.SortConfig{{ID: "latest", Sort: "registeredAt"}},
	}

	err := agg.Update(context.Background(), "test_feed", feedCfg)
	if err == nil {
		t.Fatal("expected error on fetch failure")
	}

	got, ok := cache.Get("test_feed")
	if !ok {
		t.Fatal("regression: cache must not be cleared on fetch error")
	}
	if len(got.Videos) != 2 {
		t.Errorf("regression: old cache should have 2 videos, got %d", len(got.Videos))
	}
	if got.ETag != `W/"old-etag"` {
		t.Errorf("regression: ETag should be preserved, got '%s'", got.ETag)
	}
}
