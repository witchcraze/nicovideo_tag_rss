package nico

import (
	"context"
	"errors"
	"io"
	"net/http"
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

// MockRoundTripper for testing retry logic
type MockRoundTripper struct {
	callCount int
	failTimes int // Number of times to fail before succeeding
	statusCode int
	body string
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
