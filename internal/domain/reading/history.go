// Package reading defines core reading models.
package reading

import "time"

// HistoryItem represents an item in the read history/cache.
// It mirrors Item but adds tracking fields.
type HistoryItem struct {
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

// History holds cached items keyed by GUID.
type History struct {
	items map[string]*HistoryItem
}

// NewHistory constructs a History instance from an optional item map.
func NewHistory(items map[string]*HistoryItem) *History {
	if items == nil {
		items = make(map[string]*HistoryItem)
	}
	return &History{items: items}
}

// Items returns the underlying map for read-only iteration.
func (h *History) Items() map[string]*HistoryItem {
	return h.items
}

// Snapshot returns a slice snapshot of all items.
func (h *History) Snapshot() []*HistoryItem {
	items := make([]*HistoryItem, 0, len(h.items))
	for _, v := range h.items {
		items = append(items, v)
	}
	return items
}

// MergeFeed merges a fetched feed into history.
func (h *History) MergeFeed(feed *Feed, savedAt time.Time) {
	if feed == nil {
		return
	}
	for _, it := range feed.Items {
		guid := it.GUID
		if guid == "" {
			guid = it.Link
		}
		if guid == "" {
			guid = it.Title
		}

		if existing, exists := h.items[guid]; exists {
			if existing.FeedURL == "" {
				existing.FeedURL = it.FeedURL
			}
			continue
		}

		h.items[guid] = &HistoryItem{
			GUID:        guid,
			Title:       it.Title,
			Description: it.Description,
			Content:     it.Content,
			Link:        it.Link,
			Published:   it.Published,
			Date:        it.Date,
			FeedTitle:   it.FeedTitle,
			FeedURL:     it.FeedURL,
			IsRead:      false,
			SavedAt:     savedAt,
		}
	}
}

// MarkRead marks an item as read. Returns true if it existed.
func (h *History) MarkRead(guid string) bool {
	item, ok := h.items[guid]
	if !ok {
		return false
	}
	item.IsRead = true
	return true
}

// ItemsByFeed returns history items filtered by feed URL.
func (h *History) ItemsByFeed(feedURL string) []*HistoryItem {
	items := make([]*HistoryItem, 0, len(h.items))
	for _, hItem := range h.items {
		if feedURL == AllFeedsURL || hItem.FeedURL == feedURL {
			items = append(items, hItem)
		}
	}
	return items
}
