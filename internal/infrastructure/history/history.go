// Package history manages persistent storage of feed item states.
package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/tesso57/reazy/internal/domain/reading"
)

// Manager handles loading and saving history.
type Manager struct {
	mu   sync.RWMutex
	path string
}

// NewManager creates a new history manager.
func NewManager(path string) *Manager {
	return new(Manager{
		path: path,
	})
}

// Load reads the JSONL file and returns a map of GUID -> HistoryItem.
func (m *Manager) Load() (map[string]*reading.HistoryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make(map[string]*reading.HistoryItem)
	f, err := os.Open(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return items, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var item reading.HistoryItem
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			continue // Skip malformed lines
		}
		items[item.GUID] = &item
	}

	return items, scanner.Err()
}

// Save writes the given items to the JSONL file, overwriting it.
func (m *Manager) Save(items []*reading.HistoryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	f, err := os.Create(m.path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}
