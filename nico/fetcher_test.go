package nico

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
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
	videos, err := fetcher.FetchByTag(context.Background(), "paula", "registeredAt")
	if err != nil {
		t.Fatalf("FetchByTag failed: %v", err)
	}
	if len(videos) == 0 {
		t.Fatal("expected videos to be fetched, got 0")
	}
}

// MockPaginationRoundTripper simulates paginated responses
type MockPaginationRoundTripper struct {
	callCount int
	lastSort  string
}

func (m *MockPaginationRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.callCount++
	m.lastSort = req.URL.Query().Get("sort")

	// Extract page number from query
	page := req.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

	// Return different HTML based on page number (simplified mock)
	var body string
	switch page {
	case "1":
		body = `<meta name="server-response" content="{&quot;data&quot;:{&quot;response&quot;:{&quot;$getSearchVideoV2&quot;:{&quot;data&quot;:{&quot;items&quot;:[{&quot;id&quot;:&quot;sm1&quot;,&quot;title&quot;:&quot;Video1&quot;,&quot;registeredAt&quot;:&quot;2026-01-01T00:00:00+09:00&quot;,&quot;shortDescription&quot;:&quot;desc1&quot;,&quot;thumbnail&quot;:{&quot;url&quot;:&quot;http://thumb1.jpg&quot;},&quot;owner&quot;:{&quot;name&quot;:&quot;author1&quot;}}]}}}}}"/>`
	case "2":
		body = `<meta name="server-response" content="{&quot;data&quot;:{&quot;response&quot;:{&quot;$getSearchVideoV2&quot;:{&quot;data&quot;:{&quot;items&quot;:[{&quot;id&quot;:&quot;sm2&quot;,&quot;title&quot;:&quot;Video2&quot;,&quot;registeredAt&quot;:&quot;2026-01-01T00:00:00+09:00&quot;,&quot;shortDescription&quot;:&quot;desc2&quot;,&quot;thumbnail&quot;:{&quot;url&quot;:&quot;http://thumb2.jpg&quot;},&quot;owner&quot;:{&quot;name&quot;:&quot;author2&quot;}}]}}}}}"/>`
	case "3":
		body = `<meta name="server-response" content="{&quot;data&quot;:{&quot;response&quot;:{&quot;$getSearchVideoV2&quot;:{&quot;data&quot;:{&quot;items&quot;:[{&quot;id&quot;:&quot;sm3&quot;,&quot;title&quot;:&quot;Video3&quot;,&quot;registeredAt&quot;:&quot;2026-01-01T00:00:00+09:00&quot;,&quot;shortDescription&quot;:&quot;desc3&quot;,&quot;thumbnail&quot;:{&quot;url&quot;:&quot;http://thumb3.jpg&quot;},&quot;owner&quot;:{&quot;name&quot;:&quot;author3&quot;}}]}}}}}"/>`
	default:
		// Empty result for pages beyond 3
		body = `<meta name="server-response" content="{&quot;data&quot;:{&quot;response&quot;:{&quot;$getSearchVideoV2&quot;:{&quot;data&quot;:{&quot;items&quot;:[]}}}}}"/>`
	}

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

// MockRoundTripper for testing retry logic
type MockRoundTripper struct {
	callCount  int
	failTimes  int // Number of times to fail before succeeding
	statusCode int
	shouldFail bool // When true, always fail
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.callCount++

	// Count down failures
	if m.callCount <= m.failTimes {
		if m.shouldFail && m.callCount == m.failTimes {
			return nil, errors.New("simulated network error")
		}
		// Return server error for retryable status codes
		return &http.Response{
			StatusCode: m.statusCode,
			Body:       io.NopCloser(nil),
		}, nil
	}

	// Success response
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(nil),
	}, nil
}

func TestRetryableClient_Success(t *testing.T) {
	mock := &MockRoundTripper{failTimes: 0, statusCode: 500}
	client := &http.Client{Transport: mock}
	retrier := NewRetryableClient(client)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := retrier.Do(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount)
	}
}

func TestRetryableClient_RetryOnServerError(t *testing.T) {
	// Fail 2 times with 500, then succeed
	mock := &MockRoundTripper{failTimes: 2, statusCode: 500}
	client := &http.Client{Transport: mock}
	retrier := NewRetryableClient(client)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	start := time.Now()
	resp, err := retrier.Do(context.Background(), req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.callCount)
	}

	// Check that backoff was applied (should be at least 1 second)
	if elapsed < 1*time.Second {
		t.Errorf("expected at least 1 second delay, got %v", elapsed)
	}
}

func TestRetryableClient_MaxRetriesExceeded(t *testing.T) {
	// Always fail
	mock := &MockRoundTripper{failTimes: 10, statusCode: 500}
	client := &http.Client{Transport: mock}
	retrier := NewRetryableClient(client)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := retrier.Do(context.Background(), req)

	if err == nil {
		t.Error("expected error after max retries, got nil")
	}
	if mock.callCount != 5 {
		t.Errorf("expected 5 calls (1 initial + 4 retries), got %d", mock.callCount)
	}
}

func TestRetryableClient_MaxRetriesExceededWithResponse(t *testing.T) {
	// Always return 500 (resp is non-nil). This covers the `resp != nil` branch
	// in Do() when max retries is exceeded with a server error response.
	mock := &MockRoundTripper{failTimes: 10, statusCode: 500, shouldFail: false}
	client := &http.Client{Transport: mock}
	retrier := NewRetryableClient(client)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := retrier.Do(context.Background(), req)

	if err == nil {
		t.Error("expected error after max retries, got nil")
	}
	if !strings.Contains(err.Error(), "max retries exceeded: status 500") {
		t.Errorf("expected error to contain 'max retries exceeded: status 500', got %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response when max retries exceeded with 5xx")
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
	if mock.callCount != 5 {
		t.Errorf("expected 5 calls (1 initial + 4 retries), got %d", mock.callCount)
	}
}

func TestRetryableClient_NoRetryOn200(t *testing.T) {
	mock := &MockRoundTripper{statusCode: 200}
	client := &http.Client{Transport: mock}
	retrier := NewRetryableClient(client)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := retrier.Do(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 call (no retries), got %d", mock.callCount)
	}
}

func TestRetryableClient_MinimumInterval(t *testing.T) {
	// Consecutive requests should have at least 1 second interval
	mock := &MockRoundTripper{failTimes: 0, statusCode: 200}
	client := &http.Client{Transport: mock}
	retrier := NewRetryableClient(client)

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// First request
	_, err := retrier.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	// Second request - should wait at least 1 second
	start2 := time.Now()
	_, err = retrier.Do(context.Background(), req)
	elapsed2 := time.Since(start2)

	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	// The second request should have waited at least 1 second
	if elapsed2 < 1*time.Second {
		t.Errorf("expected at least 1 second delay before second request, got %v", elapsed2)
	}
}

func TestFetchByTag_SinglePage(t *testing.T) {
	mock := &MockPaginationRoundTripper{}
	client := &http.Client{Transport: mock}
	fetcher := &htmlFetcher{
		client: NewRetryableClient(client),
	}
	fetcher.maxPages = 1

	videos, err := fetcher.FetchByTag(context.Background(), "test", "viewCount")
	if err != nil {
		t.Fatalf("FetchByTag failed: %v", err)
	}

	// Should fetch only page 1
	if len(videos) != 1 {
		t.Errorf("expected 1 video, got %d", len(videos))
	}
	if videos[0].ID != "sm1" {
		t.Errorf("expected first video ID 'sm1', got '%s'", videos[0].ID)
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mock.callCount)
	}
	if mock.lastSort != "viewCount" {
		t.Errorf("expected sort 'viewCount', got '%s'", mock.lastSort)
	}
}

func TestFetchByTag_MultiplePages(t *testing.T) {
	mock := &MockPaginationRoundTripper{}
	client := &http.Client{Transport: mock}
	fetcher := &htmlFetcher{
		client: NewRetryableClient(client),
	}
	fetcher.maxPages = 3

	videos, err := fetcher.FetchByTag(context.Background(), "test", "registeredAt")
	if err != nil {
		t.Fatalf("FetchByTag failed: %v", err)
	}

	// Should fetch 3 pages = 3 videos
	if len(videos) != 3 {
		t.Errorf("expected 3 videos, got %d", len(videos))
	}
	if videos[0].ID != "sm1" || videos[1].ID != "sm2" || videos[2].ID != "sm3" {
		t.Errorf("unexpected video IDs after pagination")
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 HTTP calls, got %d", mock.callCount)
	}
	if mock.lastSort != "registeredAt" {
		t.Errorf("expected sort 'registeredAt', got '%s'", mock.lastSort)
	}
}

func TestFetchByTag_DefaultSinglePage(t *testing.T) {
	mock := &MockPaginationRoundTripper{}
	client := &http.Client{Transport: mock}
	fetcher := &htmlFetcher{
		client: NewRetryableClient(client),
	}
	// maxPages not set, should default to 1

	videos, err := fetcher.FetchByTag(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("FetchByTag failed: %v", err)
	}

	// Should fetch only page 1 by default
	if len(videos) != 1 {
		t.Errorf("expected 1 video (default 1 page), got %d", len(videos))
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", mock.callCount)
	}
	if mock.lastSort != "registeredAt" {
		t.Errorf("expected sort 'registeredAt' (default), got '%s'", mock.lastSort)
	}
}

func TestRetryableClient_NoRetryOn4xx(t *testing.T) {
	mock := &MockRoundTripper{statusCode: 404, failTimes: 10}
	client := &http.Client{Transport: mock}
	retrier := NewRetryableClient(client)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := retrier.Do(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.callCount)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 status, got %d", resp.StatusCode)
	}
}

func TestParseHTML_NoMetaTag(t *testing.T) {
	html := strings.NewReader("<html><body></body></html>")
	fetcher := NewHTMLFetcher()
	_, err := fetcher.parseHTML(html)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error containing 'not found', got %v", err)
	}
}

func TestParseHTML_InvalidJSON(t *testing.T) {
	html := strings.NewReader(`<meta name="server-response" content="not-json"/>`)
	fetcher := NewHTMLFetcher()
	_, err := fetcher.parseHTML(html)
	if err == nil || !strings.Contains(err.Error(), "failed to parse json") {
		t.Errorf("expected error containing 'failed to parse json', got %v", err)
	}
}

func TestParseHTML_InvalidDate(t *testing.T) {
	html := strings.NewReader(`<meta name="server-response" content="{&quot;data&quot;:{&quot;response&quot;:{&quot;$getSearchVideoV2&quot;:{&quot;data&quot;:{&quot;items&quot;:[{&quot;id&quot;:&quot;sm1&quot;,&quot;title&quot;:&quot;Video1&quot;,&quot;registeredAt&quot;:&quot;invalid-date&quot;,&quot;shortDescription&quot;:&quot;desc1&quot;,&quot;thumbnail&quot;:{&quot;url&quot;:&quot;http://thumb1.jpg&quot;},&quot;owner&quot;:{&quot;name&quot;:&quot;author1&quot;}}]}}}}}"/>`)
	fetcher := NewHTMLFetcher()
	videos, err := fetcher.parseHTML(html)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}
	// The fallback is time.Now(), so we check if it's close to current time
	if time.Since(videos[0].PubDate) > time.Minute {
		t.Errorf("expected PubDate to be close to now, got %v", videos[0].PubDate)
	}
}

func TestFetchByTag_Non200Status(t *testing.T) {
	// 403 は 4xx なので RetryableClient はリトライせず、fetchPage の非200チェックに到達してエラーになる
	mock := &MockRoundTripper{statusCode: 403, failTimes: 10}
	client := &http.Client{Transport: mock}
	fetcher := &htmlFetcher{
		client:   NewRetryableClient(client),
		maxPages: 1,
	}

	_, err := fetcher.FetchByTag(context.Background(), "tag", "registeredAt")
	if err == nil {
		t.Error("expected error on 403 status, got nil")
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 HTTP call (no retry on 4xx), got %d", mock.callCount)
	}
}
