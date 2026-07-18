package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadConfig_DefaultValues は、必須項目のみ設定された場合に
// デフォルト値が正しく適用されることを回帰テストとして固定する。
func TestLoadConfig_DefaultValues(t *testing.T) {
	yamlContent := []byte(`
feeds:
  - name: test
    tags: ["tag1"]
`)
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Listen != ":8080" {
		t.Errorf("regression: default Listen should be ':8080', got '%s'", cfg.Listen)
	}
	if cfg.UpdateInterval != 180*time.Minute {
		t.Errorf("regression: default UpdateInterval should be 180m, got %v", cfg.UpdateInterval)
	}
	if cfg.CacheDir != "./cache" {
		t.Errorf("regression: default CacheDir should be './cache', got '%s'", cfg.CacheDir)
	}
	if cfg.VideoRetentionDays != 7 {
		t.Errorf("regression: default VideoRetentionDays should be 7, got %d", cfg.VideoRetentionDays)
	}
	if cfg.MaxPages != 1 {
		t.Errorf("regression: default MaxPages should be 1, got %d", cfg.MaxPages)
	}
}

// TestLoadConfig_UpdateIntervalMinEnforced は、update_interval が最小値(60m)
// 未満の場合に60mにクランプされることを確認する回帰テスト。
func TestLoadConfig_UpdateIntervalMinEnforced(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		want     time.Duration
	}{
		{
			name:     "59m should be clamped to 60m",
			interval: "59m",
			want:     60 * time.Minute,
		},
		{
			name:     "exactly 60m should not be changed",
			interval: "60m",
			want:     60 * time.Minute,
		},
		{
			name:     "120m should remain 120m",
			interval: "120m",
			want:     120 * time.Minute,
		},
		{
			name:     "1m (way below min) should be clamped to 60m",
			interval: "1m",
			want:     60 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := []byte("update_interval: " + tt.interval + "\nfeeds:\n  - name: test\n    tags: [\"tag1\"]\n")
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configFile, yamlContent, 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			cfg, err := LoadConfig(configFile)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}
			if cfg.UpdateInterval != tt.want {
				t.Errorf("regression: interval %s => got %v, want %v", tt.interval, cfg.UpdateInterval, tt.want)
			}
		})
	}
}

// TestLoadConfig_VideoRetentionDaysZeroFallback は、video_retention_days=0
// (または負の値) の場合に7日にフォールバックされることを回帰テストとして固定する。
func TestLoadConfig_VideoRetentionDaysZeroFallback(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"zero", "0"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := []byte("video_retention_days: " + tt.value + "\nfeeds:\n  - name: test\n    tags: [\"tag1\"]\n")
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configFile, yamlContent, 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			cfg, err := LoadConfig(configFile)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}
			if cfg.VideoRetentionDays != 7 {
				t.Errorf("regression: video_retention_days=%s => got %d, want 7", tt.value, cfg.VideoRetentionDays)
			}
		})
	}
}

// TestLoadConfig_MaxPagesZeroFallback は、max_pages=0 (または負の値) の場合に
// 1にフォールバックされることを回帰テストとして固定する。
func TestLoadConfig_MaxPagesZeroFallback(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"zero", "0"},
		{"negative", "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent := []byte("max_pages: " + tt.value + "\nfeeds:\n  - name: test\n    tags: [\"tag1\"]\n")
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configFile, yamlContent, 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			cfg, err := LoadConfig(configFile)
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}
			if cfg.MaxPages != 1 {
				t.Errorf("regression: max_pages=%s => got %d, want 1", tt.value, cfg.MaxPages)
			}
		})
	}
}
