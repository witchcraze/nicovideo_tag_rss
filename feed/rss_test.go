package feed

import (
	"strings"
	"testing"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/nico"
)

func TestGenerateRSS_EmptyVideos(t *testing.T) {
	cfg := config.FeedConfig{
		Name:        "test_feed",
		Title:       "Test Feed",
		Description: "Test description",
	}

	rssGen := NewRSSGenerator()
	xml, err := rssGen.Generate(cfg, []nico.Video{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(string(xml), "<channel>") {
		t.Error("expected XML to contain '<channel>'")
	}
	if !strings.Contains(string(xml), "Test Feed") {
		t.Error("expected XML to contain feed title")
	}
}

func TestGenerateRSS_WithVideos(t *testing.T) {
	cfg := config.FeedConfig{
		Name:        "test_feed",
		Title:       "Test Feed",
		Description: "Test description",
	}
	pubDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	videos := []nico.Video{
		{
			ID:          "sm1",
			Title:       "テスト動画",
			Link:        "https://www.nicovideo.jp/watch/sm1",
			Author:      "author1",
			Description: "動画の説明",
			Thumbnail:   "https://thumb.nicovideo.jp/sm1.jpg",
			PubDate:     pubDate,
		},
	}

	rssGen := NewRSSGenerator()
	xml, err := rssGen.Generate(cfg, videos)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	body := string(xml)
	if !strings.Contains(body, "テスト動画") {
		t.Error("expected XML to contain video title")
	}
	if !strings.Contains(body, "https://www.nicovideo.jp/watch/sm1") {
		t.Error("expected XML to contain video link")
	}
	if !strings.Contains(body, "author1") {
		t.Error("expected XML to contain author name")
	}
}
