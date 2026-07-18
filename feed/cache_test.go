package feed

import (
	"os"
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

func TestDumpAndLoadFromFile(t *testing.T) {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "cache_test_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Create cache with test data
	cache := NewCache()
	now := time.Now()
	cf := &CachedFeed{
		Videos: []nico.Video{
			{ID: "sm123", Title: "Test Video 1", Link: "https://test.jp/1", PubDate: now},
			{ID: "sm456", Title: "Test Video 2", Link: "https://test.jp/2", PubDate: now.Add(-48 * time.Hour)},
		},
		RSSXML:      []byte("<rss></rss>"),
		LastUpdated: now,
		ETag:        "W/\"test\"",
	}
	cache.Set("feed1", cf)

	// Dump to file
	if err := cache.DumpToFile(tmpfile.Name()); err != nil {
		t.Fatalf("DumpToFile failed: %v", err)
	}

	// Load into new cache
	cache2 := NewCache()
	if err := cache2.LoadFromFile(tmpfile.Name()); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify
	got, ok := cache2.Get("feed1")
	if !ok {
		t.Fatal("expected feed1 to exist after load")
	}

	if len(got.Videos) != 2 {
		t.Errorf("expected 2 videos, got %d", len(got.Videos))
	}

	if got.Videos[0].ID != "sm123" || got.Videos[1].ID != "sm456" {
		t.Error("video IDs don't match after load")
	}

	if got.ETag != "W/\"test\"" {
		t.Errorf("expected ETag W/\"test\", got %s", got.ETag)
	}
}

func TestCleanExpired(t *testing.T) {
	cache := NewCache()
	now := time.Now()

	cf := &CachedFeed{
		Videos: []nico.Video{
			{ID: "sm1", PubDate: now},                           // Today
			{ID: "sm2", PubDate: now.Add(-3 * 24 * time.Hour)},  // 3 days ago
			{ID: "sm3", PubDate: now.Add(-8 * 24 * time.Hour)},  // 8 days ago (should be removed)
			{ID: "sm4", PubDate: now.Add(-15 * 24 * time.Hour)}, // 15 days ago (should be removed)
		},
		RSSXML:      []byte("<rss></rss>"),
		LastUpdated: now,
		ETag:        "test",
	}
	cache.Set("feed1", cf)

	// Clean with 7 day retention
	cache.CleanExpired("feed1", 7)

	got, ok := cache.Get("feed1")
	if !ok {
		t.Fatal("expected feed1 to exist after cleanup")
	}

	if len(got.Videos) != 2 {
		t.Errorf("expected 2 videos after cleanup, got %d", len(got.Videos))
	}

	// Verify only recent videos remain
	for _, v := range got.Videos {
		if v.ID != "sm1" && v.ID != "sm2" {
			t.Errorf("unexpected video ID after cleanup: %s", v.ID)
		}
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	cache := NewCache()
	err := cache.LoadFromFile("/nonexistent/path/cache.json")
	if err == nil {
		t.Error("expected error when loading from nonexistent file")
	}
}

func TestDumpToFileMalformed(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "cache_test_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Write malformed JSON
	if err := os.WriteFile(tmpfile.Name(), []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Try to load malformed file
	cache := NewCache()
	err = cache.LoadFromFile(tmpfile.Name())
	if err == nil {
		t.Error("expected error when loading malformed JSON")
	}
}

func TestDumpToFile_WriteError(t *testing.T) {
	cache := NewCache()
	// 存在しないディレクトリへの書き込みでエラーになることを確認
	err := cache.DumpToFile("/nonexistent/dir/cache.json")
	if err == nil {
		t.Error("expected error when writing to nonexistent path")
	}
}

func TestCleanExpired_NonexistentFeed(t *testing.T) {
	cache := NewCache()
	// パニックせず正常終了することを確認（戻り値なし）
	cache.CleanExpired("nonexistent_feed", 7)
}
