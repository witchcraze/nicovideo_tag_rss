package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/witchcraze/nicovideo_tag_rss/config"
	"github.com/witchcraze/nicovideo_tag_rss/feed"
	"github.com/witchcraze/nicovideo_tag_rss/nico"
	"github.com/witchcraze/nicovideo_tag_rss/server"
)

func main() {
	// 1. Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Parse flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// 3. Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("configuration loaded", "listen", cfg.Listen, "update_interval", cfg.UpdateInterval, "cache_dir", cfg.CacheDir, "video_retention_days", cfg.VideoRetentionDays)

	// 4. Ensure cache directory exists
	if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
		slog.Error("failed to create cache directory", "path", cfg.CacheDir, "error", err)
		os.Exit(1)
	}

	// 5. Setup dependencies
	cache := feed.NewCache()
	cacheFilePath := cfg.CacheDir + "/cache.json"

	// Load cache from file if it exists
	if err := cache.LoadFromFile(cacheFilePath); err != nil {
		slog.Warn("failed to load cache from file", "path", cacheFilePath, "error", err)
	} else {
		slog.Info("cache loaded from file", "path", cacheFilePath)
	}

	fetcher := nico.NewHTMLFetcher()
	fetcher.SetMaxPages(cfg.MaxPages)
	aggregator := feed.NewAggregator(fetcher, cache, nil)

	// 6. Start background updater
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		updateFeeds(ctx, cfg, aggregator, cache)
		ticker := time.NewTicker(cfg.UpdateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("stopping background updater")
				return
			case <-ticker.C:
				updateFeeds(ctx, cfg, aggregator, cache)
			}
		}
	}()

	// 7. Setup HTTP server
	srvHandler := server.NewHandler(cache, cfg)
	mux := http.NewServeMux()
	srvHandler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    cfg.Listen,
		Handler: mux,
	}

	// 8. Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("starting HTTP server", "address", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 9. Graceful shutdown on signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	slog.Info("shutting down gracefully...")

	// Cancel background updater
	cancel()

	// Shutdown HTTP server with a timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}

	wg.Wait()

	// Save cache to file before exiting
	if err := cache.DumpToFile(cacheFilePath); err != nil {
		slog.Error("failed to save cache to file", "path", cacheFilePath, "error", err)
	} else {
		slog.Info("cache saved to file", "path", cacheFilePath)
	}

	slog.Info("shutdown complete")
}

func updateFeeds(ctx context.Context, cfg *config.Config, aggregator *feed.Aggregator, cache *feed.Cache) {
	for _, feedCfg := range cfg.Feeds {
		slog.Info("started feed update", "feed", feedCfg.Name)
		start := time.Now()

		err := aggregator.Update(ctx, feedCfg.Name, feedCfg)
		duration := time.Since(start)

		if err != nil {
			slog.Error("feed update failed", "feed", feedCfg.Name, "duration_ms", duration.Milliseconds(), "error", err)
		} else {
			slog.Info("feed update completed", "feed", feedCfg.Name, "duration_ms", duration.Milliseconds())
		}

		// Clean expired videos from the feed
		cache.CleanExpired(feedCfg.Name, cfg.VideoRetentionDays)
	}
}
