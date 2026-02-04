package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

func TestHistoryManager(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	m := NewManager(path)

	// Test failing load on non-existent file
	items, err := m.Load()
	if err != nil {
		t.Fatalf("Load should not fail on missing file: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}

	// Test Save
	now := time.Now()
	testItems := []*reading.HistoryItem{
		{
			GUID:    "id1",
			Title:   "Title 1",
			IsRead:  true,
			SavedAt: now,
		},
		{
			GUID:    "id2",
			Title:   "Title 2",
			IsRead:  false,
			SavedAt: now,
		},
	}

	if err := m.Save(testItems); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Test Load again
	loadedItems, err := m.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loadedItems) != 2 {
		t.Errorf("Expected 2 items, got %d", len(loadedItems))
	}

	if item, ok := loadedItems["id1"]; !ok {
		t.Error("Item id1 missing")
	} else {
		if item.Title != "Title 1" {
			t.Errorf("Details mismatch: %v", item)
		}
		if !item.IsRead {
			t.Error("IsRead mismatch for id1")
		}
	}
}

func TestHistoryManager_CreateDir(t *testing.T) {
	tmpDir := t.TempDir()
	// path in non-existent subdirectory
	path := filepath.Join(tmpDir, "subdir", "history.jsonl")

	m := NewManager(path)
	items := []*reading.HistoryItem{{GUID: "1", Title: "Test"}}

	if err := m.Save(items); err != nil {
		t.Fatalf("Save should succeed even if subdir missing: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}
