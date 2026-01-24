package history

import (
	"path/filepath"
	"testing"
	"time"
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
	testItems := []*Item{
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
