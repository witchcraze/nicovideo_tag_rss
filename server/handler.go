package server

import (
	"fmt"
	"net/http"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/feed"
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

// RegisterRoutes registers endpoints on the provided ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("GET /feed/{name}", h.handleFeed)
	mux.HandleFunc("GET /", h.handleIndex)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) handleFeed(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	cf, ok := h.cache.Get(name)
	if !ok || cf == nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	if match := r.Header.Get("If-None-Match"); match != "" {
		if match == cf.ETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

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
