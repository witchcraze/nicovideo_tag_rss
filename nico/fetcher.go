package nico

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// RetryableClient wraps an HTTP client with retry logic using exponential backoff
type RetryableClient struct {
	client          *http.Client
	maxRetries      int           // Maximum number of retries (total attempts = maxRetries + 1)
	minInterval     time.Duration // Minimum interval between requests
	initialBackoff  time.Duration // Initial backoff duration
	lastRequestTime time.Time
}

// NewRetryableClient creates a new RetryableClient with default settings
func NewRetryableClient(client *http.Client) *RetryableClient {
	return &RetryableClient{
		client:         client,
		maxRetries:     4,               // 4 retries = 5 total attempts
		minInterval:    1 * time.Second, // Minimum 1 second between requests
		initialBackoff: 1 * time.Second, // Start with 1 second backoff
	}
}

// isRetryable determines if an error should trigger a retry
func (r *RetryableClient) isRetryable(resp *http.Response, err error) bool {
	if err != nil {
		// Network errors are retryable
		return true
	}
	// Retry on server errors (5xx)
	return resp.StatusCode >= 500 && resp.StatusCode < 600
}

// Do executes the request with retry logic
func (r *RetryableClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	backoff := r.initialBackoff

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		// Apply minimum interval between requests (rate limiting)
		if !r.lastRequestTime.IsZero() {
			timeSinceLastRequest := time.Since(r.lastRequestTime)
			if timeSinceLastRequest < r.minInterval {
				waitTime := r.minInterval - timeSinceLastRequest
				time.Sleep(waitTime)
			}
		}

		r.lastRequestTime = time.Now()

		// Execute request
		resp, err := r.client.Do(req)

		if err == nil && !r.isRetryable(resp, err) {
			// Success or non-retryable error
			return resp, nil
		}

		lastErr = err

		// If this was the last attempt, return the error
		if attempt == r.maxRetries {
			if resp != nil {
				return resp, fmt.Errorf("max retries exceeded: status %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
		}

		// Wait before retrying (exponential backoff)
		time.Sleep(backoff)
		backoff *= 2 // Double the backoff for next iteration
	}

	return nil, lastErr
}

// Video represents a parsed video from Nicovideo.
type Video struct {
	ID          string
	Title       string
	Link        string
	Description string
	PubDate     time.Time
	Thumbnail   string
	Author      string
}

// VideoFetcher defines the interface for fetching videos by tag.
type VideoFetcher interface {
	FetchByTag(ctx context.Context, tag string, sort string) ([]Video, error)
}

// htmlFetcher implements VideoFetcher by scraping the HTML search page.
type htmlFetcher struct {
	client   *RetryableClient
	maxPages int // Maximum number of pages to fetch (default: 1)
}

// NewHTMLFetcher creates a new HTML-based VideoFetcher.
func NewHTMLFetcher() *htmlFetcher {
	baseClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &htmlFetcher{
		client:   NewRetryableClient(baseClient),
		maxPages: 1,
	}
}

// SetMaxPages sets the maximum number of pages to fetch
func (f *htmlFetcher) SetMaxPages(maxPages int) {
	if maxPages < 1 {
		maxPages = 1
	}
	f.maxPages = maxPages
}

// FetchByTag fetches videos for a given tag across multiple pages.
func (f *htmlFetcher) FetchByTag(ctx context.Context, tag string, sort string) ([]Video, error) {
	var allVideos []Video
	maxPages := f.maxPages
	if maxPages < 1 {
		maxPages = 1
	}

	for page := 1; page <= maxPages; page++ {
		videos, err := f.fetchPage(ctx, tag, sort, page)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		if len(videos) == 0 {
			// Stop if no more videos on this page
			break
		}

		allVideos = append(allVideos, videos...)
	}

	return allVideos, nil
}

// fetchPage fetches videos from a specific page.
func (f *htmlFetcher) fetchPage(ctx context.Context, tag string, sort string, page int) ([]Video, error) {
	if sort == "" {
		sort = "registeredAt"
	}
	reqURL := fmt.Sprintf("https://www.nicovideo.jp/tag/%s?sort=%s&order=desc&page=%d", url.QueryEscape(tag), url.QueryEscape(sort), page)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "nicovideo_tag_rss/0.1 (https://github.com/witchcraze/nicovideo_tag_rss)")

	resp, err := f.client.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return f.parseHTML(resp.Body)
}

// serverResponse struct represents the expected JSON structure inside the meta tag.
type serverResponse struct {
	Data struct {
		Response struct {
			GetSearchVideoV2 struct {
				Data struct {
					Items []struct {
						ID               string `json:"id"`
						Title            string `json:"title"`
						RegisteredAt     string `json:"registeredAt"`
						ShortDescription string `json:"shortDescription"`
						Thumbnail        struct {
							URL string `json:"url"`
						} `json:"thumbnail"`
						Owner struct {
							Name string `json:"name"`
						} `json:"owner"`
					} `json:"items"`
				} `json:"data"`
			} `json:"$getSearchVideoV2"`
		} `json:"response"`
	} `json:"data"`
}

func (f *htmlFetcher) parseHTML(r io.Reader) ([]Video, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}

	meta := doc.Find("meta[name='server-response']")
	content, exists := meta.Attr("content")
	if !exists {
		return nil, fmt.Errorf("meta[name='server-response'] not found")
	}

	var sResp serverResponse
	if err := json.Unmarshal([]byte(content), &sResp); err != nil {
		return nil, fmt.Errorf("failed to parse json in meta tag: %w", err)
	}

	items := sResp.Data.Response.GetSearchVideoV2.Data.Items
	videos := make([]Video, 0, len(items))

	for _, item := range items {
		pubDate, err := time.Parse(time.RFC3339, item.RegisteredAt)
		if err != nil {
			pubDate = time.Now()
		}

		videos = append(videos, Video{
			ID:          item.ID,
			Title:       item.Title,
			Link:        "https://www.nicovideo.jp/watch/" + item.ID,
			Description: item.ShortDescription,
			PubDate:     pubDate,
			Thumbnail:   item.Thumbnail.URL,
			Author:      item.Owner.Name,
		})
	}

	return videos, nil
}
