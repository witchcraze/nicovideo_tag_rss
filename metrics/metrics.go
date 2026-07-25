package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Requests
	HTTPRequestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nicovideo_rss_http_requests_total",
			Help: "Total number of HTTP requests by endpoint and status code",
		},
		[]string{"endpoint", "status"},
	)

	// HTML Parse
	HTMLParseCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nicovideo_rss_html_parse_total",
			Help: "Total number of HTML parses by status (success/failure)",
		},
		[]string{"status"}, // "success", "failure"
	)

	// Nicovideo Requests
	NicoRequestCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nicovideo_rss_nico_requests_total",
			Help: "Total number of requests to Nicovideo by status code",
		},
		[]string{"status"},
	)

	NicoRetryCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nicovideo_rss_nico_retries_total",
			Help: "Total number of retry attempts made to Nicovideo",
		},
	)

	// RSS Cache Hit/Miss
	CacheHitCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nicovideo_rss_cache_hits_total",
			Help: "Total number of cache hits and misses",
		},
		[]string{"status"}, // "hit", "miss", "not_modified"
	)

	// RSS Feed Updates
	FeedUpdateCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nicovideo_rss_feed_updates_total",
			Help: "Total number of feed updates by feed name and status",
		},
		[]string{"feed", "status"}, // "success", "failure"
	)

	FeedUpdateDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nicovideo_rss_feed_update_duration_seconds",
			Help:    "Duration of feed updates in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"feed"},
	)
)
