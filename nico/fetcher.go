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
	FetchByTag(ctx context.Context, tag string) ([]Video, error)
}

// htmlFetcher implements VideoFetcher by scraping the HTML search page.
type htmlFetcher struct {
	client *http.Client
}

// NewHTMLFetcher creates a new HTML-based VideoFetcher.
func NewHTMLFetcher() *htmlFetcher {
	return &htmlFetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchByTag fetches videos for a given tag.
func (f *htmlFetcher) FetchByTag(ctx context.Context, tag string) ([]Video, error) {
	reqURL := fmt.Sprintf("https://www.nicovideo.jp/tag/%s?sort=registeredAt&order=desc", url.QueryEscape(tag))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "nicovideo_tag_rss/0.1 (https://github.com/witchcraze/nicovideo_tag_rss)")

	resp, err := f.client.Do(req)
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
