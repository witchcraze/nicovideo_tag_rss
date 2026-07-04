package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/feed"
)

func TestHandler_Healthz(t *testing.T) {
	cache := feed.NewCache()
	h := NewHandler(cache, nil)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %v", rr.Code)
	}
	if rr.Body.String() != "OK" {
		t.Errorf("expected OK, got %v", rr.Body.String())
	}
}

func TestHandler_FeedXML(t *testing.T) {
	cache := feed.NewCache()
	cf := &feed.CachedFeed{
		RSSXML:      []byte("<rss>test</rss>"),
		ETag:        `W/"test"`,
		LastUpdated: time.Now(),
	}
	cache.Set("test_feed", cf)

	h := NewHandler(cache, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Test successful get
	req := httptest.NewRequest("GET", "/feed/test_feed", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %v", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "application/rss+xml; charset=utf-8" {
		t.Errorf("expected rss content type, got %v", rr.Header().Get("Content-Type"))
	}
	if rr.Header().Get("ETag") != `W/"test"` {
		t.Errorf("expected ETag W/\"test\", got %v", rr.Header().Get("ETag"))
	}
	if rr.Body.String() != "<rss>test</rss>" {
		t.Errorf("unexpected body: %v", rr.Body.String())
	}

	// Test not found
	reqNotFound := httptest.NewRequest("GET", "/feed/unknown", nil)
	rrNotFound := httptest.NewRecorder()
	mux.ServeHTTP(rrNotFound, reqNotFound)

	if rrNotFound.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %v", rrNotFound.Code)
	}

	// Test ETag Not Modified
	reqNotModified := httptest.NewRequest("GET", "/feed/test_feed", nil)
	reqNotModified.Header.Set("If-None-Match", `W/"test"`)
	rrNotModified := httptest.NewRecorder()
	mux.ServeHTTP(rrNotModified, reqNotModified)

	if rrNotModified.Code != http.StatusNotModified {
		t.Errorf("expected 304, got %v", rrNotModified.Code)
	}
}

func TestHandler_Index(t *testing.T) {
	cache := feed.NewCache()
	cfg := &config.Config{
		Feeds: []config.FeedConfig{
			{Name: "feed1", Title: "Feed 1"},
			{Name: "feed2", Title: "Feed 2"},
		},
	}
	h := NewHandler(cache, cfg)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %v", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "feed1") || !strings.Contains(body, "feed2") {
		t.Errorf("expected body to contain feed names, got %v", body)
	}
}
