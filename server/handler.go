package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/feed"
	"github.com/witchcraze/nicovideo_tag_rss/metrics"
)

// Handler handles HTTP requests.
type Handler struct {
	cache *feed.Cache
	cfg   *config.Config
}

// NewHandler creates a new Handler.
func NewHandler(cache *feed.Cache, cfg *config.Config) *Handler {
	return &Handler{
		cache: cache,
		cfg:   cfg,
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (h *Handler) instrument(endpoint string, f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		f(recorder, r)
		metrics.HTTPRequestCount.WithLabelValues(endpoint, fmt.Sprintf("%d", recorder.status)).Inc()
	}
}

// RegisterRoutes registers endpoints on the provided ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.instrument("healthz", h.handleHealthz))
	mux.HandleFunc("GET /feed/{name}", h.instrument("feed", h.handleFeed))
	mux.HandleFunc("GET /", h.instrument("index", h.handleIndex))
	mux.Handle("GET /metrics", promhttp.Handler())
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) handleFeed(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	name = strings.TrimSuffix(name, ".xml")

	cf, ok := h.cache.Get(name)
	if !ok || cf == nil {
		metrics.CacheHitCount.WithLabelValues("miss").Inc()
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	if match := r.Header.Get("If-None-Match"); match != "" {
		if match == cf.ETag {
			metrics.CacheHitCount.WithLabelValues("not_modified").Inc()
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	metrics.CacheHitCount.WithLabelValues("hit").Inc()
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("ETag", cf.ETag)
	w.Write(cf.RSSXML)
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		// Handle 404 for unknown paths since `GET /` acts as a catch-all if not careful,
		// though in Go 1.22 explicit exact match is supported depending on how it's matched.
		// Wait, in Go 1.22 `GET /` matches only the root, but `GET /{$}` is explicit root.
		// `GET /` is a prefix match. Let's make sure it only serves root.
		http.NotFound(w, r)
		return
	}

	if h.cfg == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("No configuration available."))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := "<html><body><h1>Available Feeds</h1><ul>"
	for _, f := range h.cfg.Feeds {
		html += fmt.Sprintf(`<li><a href="/feed/%s">%s</a> - %s</li>`, f.Name, f.Title, f.Description)
	}
	html += "</ul></body></html>"

	w.Write([]byte(html))
}
