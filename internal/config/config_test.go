package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reazy_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(cfg.Feeds) == 0 {
		t.Error("Expected default feeds, got empty")
	}
	if cfg.Feeds[0] != "https://news.ycombinator.com/rss" {
		t.Errorf("Expected default feed, got %s", cfg.Feeds[0])
	}

	if cfg.Theme.FeedName != "244" {
		t.Errorf("Expected default Theme.FeedName '244', got '%s'", cfg.Theme.FeedName)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file not created")
	}
}

func TestLoadConfig_Corrupt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reazy_test_corrupt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	_ = os.WriteFile(configPath, []byte("invalid_yaml: ["), 0644)

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for corrupt config read, got nil")
	}
}

func TestConfig_AddRemoveFeed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reazy_test_persist")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Test Add
	newFeed := "https://example.com/rss"
	err = cfg.AddFeed(newFeed)
	if err != nil {
		t.Errorf("AddFeed failed: %v", err)
	}

	if len(cfg.Feeds) != 2 {
		t.Errorf("Expected 2 feeds, got %d", len(cfg.Feeds))
	}
	if cfg.Feeds[1] != newFeed {
		t.Errorf("Expected %s, got %s", newFeed, cfg.Feeds[1])
	}

	// Verify persistence by reloading
	cfg2, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg2.Feeds) != 2 {
		t.Errorf("Persistence failed, expected 2 feeds, got %d", len(cfg2.Feeds))
	}

	// Test Remove
	err = cfg.RemoveFeed(0) // Remove default
	if err != nil {
		t.Errorf("RemoveFeed failed: %v", err)
	}

	if len(cfg.Feeds) != 1 {
		t.Errorf("Expected 1 feed, got %d", len(cfg.Feeds))
	}
	if cfg.Feeds[0] != newFeed {
		t.Errorf("Expected %s remaining, got %s", newFeed, cfg.Feeds[0])
	}

	// Test Remove Invalid
	err = cfg.RemoveFeed(99)
	if err == nil {
		t.Error("Expected error for invalid index")
	}
}
