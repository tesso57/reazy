// Package history manages persistent storage of feed item states.
package history

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Item represents an item in the history/cache.
// It mirrors feed.Item but adds tracking fields.
type Item struct {
	GUID        string    `json:"guid"` // Unique ID (Link or GUID)
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	Published   string    `json:"published"`
	Date        time.Time `json:"date"`
	FeedTitle   string    `json:"feed_title"`
	FeedURL     string    `json:"feed_url"`

	IsRead  bool      `json:"is_read"`
	SavedAt time.Time `json:"saved_at"`
}

// Manager handles loading and saving history.
type Manager struct {
	mu   sync.RWMutex
	path string
}

// NewManager creates a new history manager.
func NewManager(path string) *Manager {
	return &Manager{
		path: path,
	}
}

// Load reads the JSONL file and returns a map of GUID -> Item.
func (m *Manager) Load() (map[string]*Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make(map[string]*Item)
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
		var item Item
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			continue // Skip malformed lines
		}
		items[item.GUID] = &item
	}

	return items, scanner.Err()
}

// Save writes the given items to the JSONL file, overwriting it.
func (m *Manager) Save(items []*Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	// We assume parent dir created by main or config, but good to be safe if path is custom
	// For now, assume path is valid or already prepped.

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
