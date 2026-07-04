package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Listen         string        `yaml:"listen"`
	UpdateInterval time.Duration `yaml:"update_interval"`
	Feeds          []FeedConfig  `yaml:"feeds"`
}

// FeedConfig represents the configuration for a single RSS feed.
type FeedConfig struct {
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
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
		cfg.UpdateInterval = 5 * time.Minute
	}

	// Validate
	if len(cfg.Feeds) == 0 {
		return nil, fmt.Errorf("at least one feed must be defined")
	}

	seenNames := make(map[string]bool)
	for _, feed := range cfg.Feeds {
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
	}

	return cfg, nil
}
