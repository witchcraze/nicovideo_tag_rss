package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/feed"
	"github.com/witchcraze/nicovideo_tag_rss/nico"
	"github.com/witchcraze/nicovideo_tag_rss/server"
)

type mockRoundTripper struct {
	shouldFail bool
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.shouldFail {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	body := `<meta name="server-response" content="{&quot;data&quot;:{&quot;response&quot;:{&quot;$getSearchVideoV2&quot;:{&quot;data&quot;:{&quot;items&quot;:[{&quot;id&quot;:&quot;sm123&quot;,&quot;title&quot;:&quot;E2E Test Video&quot;,&quot;registeredAt&quot;:&quot;2026-07-25T12:00:00+09:00&quot;,&quot;shortDescription&quot;:&quot;Test Description&quot;,&quot;thumbnail&quot;:{&quot;url&quot;:&quot;http://example.com/thumb.jpg&quot;},&quot;owner&quot;:{&quot;name&quot;:&quot;Test Author&quot;}}]}}}}}"/>`

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestE2E_ApplicationFlow(t *testing.T) {
	// 1. Setup config
	cfg := &config.Config{
		Listen:             ":8080",
		UpdateInterval:     60 * time.Minute,
		CacheDir:           t.TempDir(),
		VideoRetentionDays: 7,
		MaxPages:           1,
		Feeds: []config.FeedConfig{
			{
				Name:        "testfeed",
				Title:       "Test Feed",
				Description: "Feed for E2E Test",
				Tags:        []string{"test_tag"},
				Sorts: []config.SortConfig{
					{
						ID:    "latest",
						Sort:  "registeredAt",
						Title: "最新投稿",
					},
				},
			},
		},
	}

	// 2. Setup mock fetcher
	mockRT := &mockRoundTripper{shouldFail: false}
	mockClient := &http.Client{Transport: mockRT}
	fetcher := nico.NewHTMLFetcher()
	fetcher.SetMaxPages(cfg.MaxPages)
	fetcher.SetClient(mockClient)

	// 3. Setup cache & aggregator
	cache := feed.NewCache()
	aggregator := feed.NewAggregator(fetcher, cache, nil)

	// 4. Run updateFeeds to populate cache
	ctx := context.Background()
	updateFeeds(ctx, cfg, aggregator, cache)

	// 5. Setup HTTP Server
	srvHandler := server.NewHandler(cache, cfg)
	mux := http.NewServeMux()
	srvHandler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 6. Test GET /feed/testfeed.xml (Success Case)
	resp, err := http.Get(ts.URL + "/feed/testfeed.xml")
	if err != nil {
		t.Fatalf("failed to GET feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if !strings.Contains(bodyStr, "<title>Test Feed</title>") {
		t.Errorf("feed title not found in response")
	}
	if !strings.Contains(bodyStr, "E2E Test Video") {
		t.Errorf("video title not found in response")
	}

	// 7. Test Error Case: cache protection
	// Make the fetcher fail
	mockRT.shouldFail = true
	// To speed up retries during test, we can set shorter backoff on client if we want
	// But our mockRT always fails anyway.

	// Run update again
	updateFeeds(ctx, cfg, aggregator, cache)

	// GET again, it should return the cached feed successfully
	resp2, err := http.Get(ts.URL + "/feed/testfeed.xml")
	if err != nil {
		t.Fatalf("failed to GET feed after error: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 after fetch error, got %d", resp2.StatusCode)
	}

	bodyBytes2, _ := io.ReadAll(resp2.Body)
	bodyStr2 := string(bodyBytes2)

	if !strings.Contains(bodyStr2, "E2E Test Video") {
		t.Errorf("expected video title in cache to be protected, but not found")
	}

	// 8. Test ETag / If-None-Match (304 Scenario)
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header in response")
	}
	req3, _ := http.NewRequest("GET", ts.URL+"/feed/testfeed.xml", nil)
	req3.Header.Set("If-None-Match", etag)
	resp3, err := ts.Client().Do(req3)
	if err != nil {
		t.Fatalf("failed to GET feed with If-None-Match: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotModified {
		t.Fatalf("expected status 304, got %d", resp3.StatusCode)
	}

	// 9. Test GET /metrics Scenario
	respMetrics, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed to GET /metrics: %v", err)
	}
	defer respMetrics.Body.Close()
	if respMetrics.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for /metrics, got %d", respMetrics.StatusCode)
	}
	metricsBytes, _ := io.ReadAll(respMetrics.Body)
	metricsStr := string(metricsBytes)
	if !strings.Contains(metricsStr, "nicovideo_rss_feed_updates_total") {
		t.Errorf("expected metrics to contain nicovideo_rss_feed_updates_total, but not found")
	}

	// 10. Test Cache Persistence Round-Trip Scenario
	cacheFilePath := filepath.Join(cfg.CacheDir, "cache.json")
	if err := cache.DumpToFile(cacheFilePath); err != nil {
		t.Fatalf("failed to dump cache to file: %v", err)
	}
	newCache := feed.NewCache()
	if err := newCache.LoadFromFile(cacheFilePath); err != nil {
		t.Fatalf("failed to load cache from file: %v", err)
	}
	cf3, ok := newCache.Get("testfeed")
	if !ok || cf3 == nil {
		t.Fatalf("expected to get testfeed from loaded cache")
	}
	if !strings.Contains(string(cf3.RSSXML), "E2E Test Video") {
		t.Errorf("loaded cache RSS XML does not contain expected video title")
	}
}
