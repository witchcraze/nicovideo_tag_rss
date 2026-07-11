package config

import (
	"fmt"
	"os"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// SortConfig represents a single sort variation.
type SortConfig struct {
	ID    string `yaml:"id"`
	Sort  string `yaml:"sort"`
	Title string `yaml:"title"`
}

// Config represents the application configuration.
type Config struct {
	Listen             string        `yaml:"listen"`
	UpdateInterval     time.Duration `yaml:"update_interval"`
	CacheDir           string        `yaml:"cache_dir"`
	VideoRetentionDays int           `yaml:"video_retention_days"`
	MaxPages           int           `yaml:"max_pages"`
	Feeds              []FeedConfig  `yaml:"feeds"`
}

// FeedConfig represents the configuration for a single RSS feed.
type FeedConfig struct {
	Name        string       `yaml:"name"`
	Title       string       `yaml:"title"`
	Description string       `yaml:"description"`
	Tags        []string     `yaml:"tags"`
	Sorts       []SortConfig `yaml:"sorts"`
}

// LoadConfig reads and parses the YAML configuration file, then validates it.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	// Set defaults
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.UpdateInterval == 0 {
		// Default to 180 minutes to reduce aggressive polling
		cfg.UpdateInterval = 180 * time.Minute
	}
	// Enforce a sensible minimum to protect both our service and the upstream
	if cfg.UpdateInterval < 60*time.Minute {
		cfg.UpdateInterval = 60 * time.Minute
	}
	if cfg.CacheDir == "" {
		cfg.CacheDir = "./cache"
	}
	if cfg.VideoRetentionDays <= 0 {
		cfg.VideoRetentionDays = 7
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 1
	}

	// Validate
	if len(cfg.Feeds) == 0 {
		return nil, fmt.Errorf("at least one feed must be defined")
	}

	seenNames := make(map[string]bool)
	for i := range cfg.Feeds {
		feed := &cfg.Feeds[i]
		if feed.Name == "" {
			return nil, fmt.Errorf("feed name cannot be empty")
		}
		if seenNames[feed.Name] {
			return nil, fmt.Errorf("duplicate feed name '%s'", feed.Name)
		}
		seenNames[feed.Name] = true

		if len(feed.Tags) == 0 {
			return nil, fmt.Errorf("feed '%s' must have at least one tag", feed.Name)
		}

		// Set default Sorts if empty
		if len(feed.Sorts) == 0 {
			feed.Sorts = []SortConfig{
				{
					ID:    "latest",
					Sort:  "registeredAt",
					Title: "最新投稿",
				},
			}
		}

		// Validate Sorts
		seenSortIDs := make(map[string]bool)
		for _, s := range feed.Sorts {
			if s.ID == "" {
				return nil, fmt.Errorf("feed '%s': sort ID cannot be empty", feed.Name)
			}
			if s.Sort == "" {
				return nil, fmt.Errorf("feed '%s': sort value cannot be empty", feed.Name)
			}
			if seenSortIDs[s.ID] {
				return nil, fmt.Errorf("feed '%s': duplicate sort ID '%s'", feed.Name, s.ID)
			}
			seenSortIDs[s.ID] = true
		}
	}

	return cfg, nil
}
