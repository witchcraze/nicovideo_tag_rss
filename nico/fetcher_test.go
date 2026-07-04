package nico

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNicoFetcher_Parse(t *testing.T) {
	htmlPath := "testdata/raw_paula.html"
	f, err := os.Open(htmlPath)
	if err != nil {
		t.Skipf("Skipping test because testdata is missing: %v", err)
	}
	defer f.Close()

	fetcher := NewHTMLFetcher()
	
	videos, err := fetcher.parseHTML(f)
	if err != nil {
		t.Fatalf("parseHTML failed: %v", err)
	}

	if len(videos) == 0 {
		t.Fatal("expected videos to be parsed, got 0")
	}

	v := videos[0]
	if v.ID == "" {
		t.Error("expected non-empty ID")
	}
	if v.Title == "" {
		t.Error("expected non-empty Title")
	}
	if v.Link == "" {
		t.Error("expected non-empty Link")
	}
	if v.PubDate.IsZero() {
		t.Error("expected non-zero PubDate")
	}

	// 1件目の確認 (raw_paula.html の先頭データに基づく)
	// title: Paula Vianca Lim Japan 99, id: sm33716375, registeredAt: 2018-08-19T17:20:03+09:00
	if v.ID != "sm33716375" {
		t.Errorf("expected ID 'sm33716375', got '%s'", v.ID)
	}
	if v.Title != "Paula Vianca Lim Japan 99" {
		t.Errorf("expected Title 'Paula Vianca Lim Japan 99', got '%s'", v.Title)
	}
	if v.Link != "https://www.nicovideo.jp/watch/sm33716375" {
		t.Errorf("expected Link to be 'https://www.nicovideo.jp/watch/sm33716375', got '%s'", v.Link)
	}
	
	expectedTime, _ := time.Parse(time.RFC3339, "2018-08-19T17:20:03+09:00")
	if !v.PubDate.Equal(expectedTime) {
		t.Errorf("expected PubDate '%v', got '%v'", expectedTime, v.PubDate)
	}
	if v.Author != "Paula vianca" {
		t.Errorf("expected Author 'Paula vianca', got '%s'", v.Author)
	}
	if v.Thumbnail == "" {
		t.Error("expected non-empty Thumbnail")
	}
}

func TestNicoFetcher_FetchByTag(t *testing.T) {
	// 実際の通信を伴うため、通常はスキップするか、インターフェースのモックを使います。
	// ここでは実際の FetchByTag の動作確認をローカルでのみ行うように Skip します。
	t.Skip("Skipping actual network request test")

	fetcher := NewHTMLFetcher()
	videos, err := fetcher.FetchByTag(context.Background(), "paula")
	if err != nil {
		t.Fatalf("FetchByTag failed: %v", err)
	}
	if len(videos) == 0 {
		t.Fatal("expected videos to be fetched, got 0")
	}
}
