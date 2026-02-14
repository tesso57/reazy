package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reazy_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	store, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(store.Settings.Feeds) == 0 {
		t.Error("Expected default feeds, got empty")
	}
	if store.Settings.Feeds[0] != "https://news.ycombinator.com/rss" {
		t.Errorf("Expected default feed, got %s", store.Settings.Feeds[0])
	}

	if store.Settings.Theme.FeedName != "244" {
		t.Errorf("Expected default Theme.FeedName '244', got '%s'", store.Settings.Theme.FeedName)
	}
	if store.Settings.KeyMap.Summarize != "s" {
		t.Errorf("Expected default KeyMap.Summarize 's', got %q", store.Settings.KeyMap.Summarize)
	}
	if store.Settings.KeyMap.ToggleSummary != "S" {
		t.Errorf("Expected default KeyMap.ToggleSummary 'S', got %q", store.Settings.KeyMap.ToggleSummary)
	}
	if store.Settings.Codex.Command != "codex" {
		t.Errorf("Expected default Codex.Command 'codex', got %q", store.Settings.Codex.Command)
	}
	if store.Settings.Codex.WebSearch != "disabled" {
		t.Errorf("Expected default Codex.WebSearch 'disabled', got %q", store.Settings.Codex.WebSearch)
	}
	if store.Settings.Codex.TimeoutSeconds != 30 {
		t.Errorf("Expected default Codex.TimeoutSeconds 30, got %d", store.Settings.Codex.TimeoutSeconds)
	}
	if filepath.Base(store.Settings.HistoryFile) != "history.db" {
		t.Errorf("Expected default history db path, got %q", store.Settings.HistoryFile)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file not created")
	}
}

func TestLoad_NormalizesLegacyJSONLHistoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := "history_file: " + filepath.Join(tmpDir, "history.jsonl") + "\n"
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	store, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := filepath.Join(tmpDir, "history.db")
	if store.Settings.HistoryFile != want {
		t.Fatalf("HistoryFile = %q, want %q", store.Settings.HistoryFile, want)
	}
}

func TestLoad_Corrupt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reazy_test_corrupt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	_ = os.WriteFile(configPath, []byte("invalid_yaml: ["), 0600)

	_, err = Load(configPath)
	if err == nil {
		t.Error("Expected error for corrupt config read, got nil")
	}
}

func TestStore_AddRemoveFeed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reazy_test_persist")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	store, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Test Add
	newFeed := "https://example.com/rss"
	err = store.Add(newFeed)
	if err != nil {
		t.Errorf("AddFeed failed: %v", err)
	}

	if len(store.Settings.Feeds) != 2 {
		t.Errorf("Expected 2 feeds, got %d", len(store.Settings.Feeds))
	}
	if store.Settings.Feeds[1] != newFeed {
		t.Errorf("Expected %s, got %s", newFeed, store.Settings.Feeds[1])
	}

	// Verify persistence by reloading
	store2, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(store2.Settings.Feeds) != 2 {
		t.Errorf("Persistence failed, expected 2 feeds, got %d", len(store2.Settings.Feeds))
	}

	// Test Remove
	err = store.Remove(0) // Remove default
	if err != nil {
		t.Errorf("RemoveFeed failed: %v", err)
	}

	if len(store.Settings.Feeds) != 1 {
		t.Errorf("Expected 1 feed, got %d", len(store.Settings.Feeds))
	}
	if store.Settings.Feeds[0] != newFeed {
		t.Errorf("Expected %s remaining, got %s", newFeed, store.Settings.Feeds[0])
	}

	// Test Remove Invalid
	err = store.Remove(99)
	if err == nil {
		t.Error("Expected error for invalid index")
	}
}

func TestLoad_NormalizesFeeds(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reazy_test_normalize")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	content := `feeds:
  - " https://example.com/rss "
  - |
      https://example.com/one.atom
      https://example.com/two.atom
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	store, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := []string{
		"https://example.com/rss",
		"https://example.com/one.atom",
		"https://example.com/two.atom",
	}

	if len(store.Settings.Feeds) != len(want) {
		t.Fatalf("Expected %d feeds, got %d", len(want), len(store.Settings.Feeds))
	}
	for i, got := range store.Settings.Feeds {
		if got != want[i] {
			t.Fatalf("Expected feed %d to be %q, got %q", i, want[i], got)
		}
	}
}
