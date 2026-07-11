package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// 正常系テストデータ
	validYAML := []byte(`
listen: ":8080"
update_interval: 5m

feeds:
  - name: paula
    title: パウラ動画
    description: パウラ関連動画
    tags:
      - パウラ
  - name: vtuber
    title: VTuber
    description: VTuberまとめ
    tags:
      - パウラ
      - 結月ゆかり
      - VOICEROID実況
`)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, validYAML, 0644); err != nil {
		t.Fatalf("failed to write tmp config file: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Listen != ":8080" {
		t.Errorf("expected Listen to be ':8080', got '%s'", cfg.Listen)
	}
	// The configured value 5m is below the enforced minimum (60m), so expect 60m
	expectedInterval := 60 * time.Minute
	if cfg.UpdateInterval != expectedInterval {
		t.Errorf("expected UpdateInterval to be %v, got %v", expectedInterval, cfg.UpdateInterval)
	}
	if len(cfg.Feeds) != 2 {
		t.Fatalf("expected 2 feeds, got %d", len(cfg.Feeds))
	}
	if cfg.Feeds[0].Name != "paula" {
		t.Errorf("expected feed[0].Name to be 'paula', got '%s'", cfg.Feeds[0].Name)
	}
	if cfg.Feeds[1].Name != "vtuber" {
		t.Errorf("expected feed[1].Name to be 'vtuber', got '%s'", cfg.Feeds[1].Name)
	}
	if len(cfg.Feeds[1].Tags) != 3 {
		t.Fatalf("expected 3 tags in vtuber feed, got %d", len(cfg.Feeds[1].Tags))
	}
	if cfg.Feeds[1].Tags[1] != "結月ゆかり" {
		t.Errorf("expected second tag in vtuber feed to be '結月ゆかり', got '%s'", cfg.Feeds[1].Tags[1])
	}

	// Verify default sorts are set
	for _, feed := range cfg.Feeds {
		if len(feed.Sorts) != 1 {
			t.Fatalf("expected 1 default sort, got %d", len(feed.Sorts))
		}
		s := feed.Sorts[0]
		if s.ID != "latest" {
			t.Errorf("expected default sort ID 'latest', got '%s'", s.ID)
		}
		if s.Sort != "registeredAt" {
			t.Errorf("expected default sort 'registeredAt', got '%s'", s.Sort)
		}
		if s.Title != "最新投稿" {
			t.Errorf("expected default sort title '最新投稿', got '%s'", s.Title)
		}
	}
}

func TestLoadConfig_ExplicitSorts(t *testing.T) {
	yamlContent := []byte(`
feeds:
  - name: vocaloid
    title: VOCALOID
    tags:
      - VOCALOID
    sorts:
      - id: latest
        sort: registeredAt
        title: 最新投稿
      - id: popular
        sort: viewCount
        title: 人気順
`)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write tmp config: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	feed := cfg.Feeds[0]
	if len(feed.Sorts) != 2 {
		t.Fatalf("expected 2 sorts, got %d", len(feed.Sorts))
	}

	if feed.Sorts[0].ID != "latest" || feed.Sorts[0].Sort != "registeredAt" || feed.Sorts[0].Title != "最新投稿" {
		t.Errorf("unexpected first sort: %+v", feed.Sorts[0])
	}
	if feed.Sorts[1].ID != "popular" || feed.Sorts[1].Sort != "viewCount" || feed.Sorts[1].Title != "人気順" {
		t.Errorf("unexpected second sort: %+v", feed.Sorts[1])
	}
}

func TestLoadConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErrMsg  string
	}{
		{
			name: "missing required feeds",
			yamlContent: `
listen: ":8080"
update_interval: 5m
feeds: []
`,
			wantErrMsg: "at least one feed must be defined",
		},
		{
			name: "missing feed name",
			yamlContent: `
listen: ":8080"
feeds:
  - title: タイトル
    tags: ["tag1"]
`,
			wantErrMsg: "feed name cannot be empty",
		},
		{
			name: "empty tags",
			yamlContent: `
listen: ":8080"
feeds:
  - name: test
    title: タイトル
    tags: []
`,
			wantErrMsg: "feed 'test' must have at least one tag",
		},
		{
			name: "duplicate feed name",
			yamlContent: `
listen: ":8080"
feeds:
  - name: test
    title: タイトル1
    tags: ["tag1"]
  - name: test
    title: タイトル2
    tags: ["tag2"]
`,
			wantErrMsg: "duplicate feed name 'test'",
		},
		{
			name: "empty sort ID",
			yamlContent: `
feeds:
  - name: test
    tags: ["tag1"]
    sorts:
      - id: ""
        sort: "registeredAt"
        title: "最新"
`,
			wantErrMsg: "sort ID cannot be empty",
		},
		{
			name: "empty sort value",
			yamlContent: `
feeds:
  - name: test
    tags: ["tag1"]
    sorts:
      - id: "latest"
        sort: ""
        title: "最新"
`,
			wantErrMsg: "sort value cannot be empty",
		},
		{
			name: "duplicate sort ID",
			yamlContent: `
feeds:
  - name: test
    tags: ["tag1"]
    sorts:
      - id: "latest"
        sort: "registeredAt"
        title: "最新"
      - id: "latest"
        sort: "viewCount"
        title: "人気"
`,
			wantErrMsg: "duplicate sort ID 'latest'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configFile, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("failed to write tmp config file: %v", err)
			}

			_, err := LoadConfig(configFile)
			if err == nil {
				t.Fatalf("expected error containing '%s', got nil", tt.wantErrMsg)
			}
			if err != nil && !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("expected error containing '%s', got '%v'", tt.wantErrMsg, err)
			}
		})
	}
}
