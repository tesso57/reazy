// Package reading defines core reading models.
package reading

import (
	"sort"
	"strings"
	"time"
)

// HistoryItem represents an item in the read history/cache.
// It mirrors Item but adds tracking fields.
type HistoryItem struct {
	GUID        string    `json:"guid"` // Unique ID (Link or GUID)
	Kind        string    `json:"kind,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	Published   string    `json:"published"`
	Date        time.Time `json:"date"`
	FeedTitle   string    `json:"feed_title"`
	FeedURL     string    `json:"feed_url"`

	IsRead       bool      `json:"is_read"`
	SavedAt      time.Time `json:"saved_at"`
	IsBookmarked bool      `json:"is_bookmarked"`
	AISummary    string    `json:"ai_summary,omitempty"`
	AITags       []string  `json:"ai_tags,omitempty"`
	AIUpdatedAt  time.Time `json:"ai_updated_at"`
	DigestDate   string    `json:"digest_date,omitempty"`
	RelatedGUIDs []string  `json:"related_guids,omitempty"`
	BodyHydrated bool      `json:"-"`
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
	return new(History{items: items})
}

// Items returns the underlying map for read-only iteration.
func (h *History) Items() map[string]*HistoryItem {
	return h.items
}

// Snapshot returns a slice snapshot of all items.
func (h *History) Snapshot() []*HistoryItem {
	items := make([]*HistoryItem, 0, len(h.items))
	for _, v := range h.items {
		if v == nil {
			continue
		}
		items = append(items, v)
	}
	return items
}

// MergeFeed merges a fetched feed into history.
func (h *History) MergeFeed(feed *Feed, savedAt time.Time) []*HistoryItem {
	if feed == nil {
		return nil
	}
	changed := make([]*HistoryItem, 0, len(feed.Items))
	for _, it := range feed.Items {
		guid := it.GUID
		if guid == "" {
			guid = it.Link
		}
		if guid == "" {
			guid = it.Title
		}
		if strings.TrimSpace(guid) == "" {
			continue
		}

		if existing, exists := h.items[guid]; exists {
			if existing == nil || existing.kind() == NewsDigestKind {
				continue
			}
			if mergeFetchedArticle(existing, it, savedAt) {
				changed = append(changed, existing)
			}
			continue
		}

		newItem := &HistoryItem{
			GUID:         guid,
			Kind:         ArticleKind,
			Title:        it.Title,
			Description:  it.Description,
			Content:      it.Content,
			Link:         it.Link,
			Published:    it.Published,
			Date:         it.Date,
			FeedTitle:    it.FeedTitle,
			FeedURL:      it.FeedURL,
			IsRead:       false,
			SavedAt:      savedAt,
			BodyHydrated: true,
		}
		h.items[guid] = newItem
		changed = append(changed, newItem)
	}
	return changed
}

func mergeFetchedArticle(existing *HistoryItem, fetched Item, savedAt time.Time) bool {
	if existing == nil {
		return false
	}

	changed := false
	if existing.Kind == "" {
		existing.Kind = ArticleKind
		changed = true
	}
	if fetched.Title != "" && fetched.Title != existing.Title {
		existing.Title = fetched.Title
		changed = true
	}
	if fetched.Description != "" && fetched.Description != existing.Description {
		existing.Description = fetched.Description
		changed = true
	}
	if fetched.Content != existing.Content {
		existing.Content = fetched.Content
		changed = true
	}
	if fetched.Link != "" && fetched.Link != existing.Link {
		existing.Link = fetched.Link
		changed = true
	}
	if fetched.Published != "" && fetched.Published != existing.Published {
		existing.Published = fetched.Published
		changed = true
	}
	if !fetched.Date.IsZero() && !fetched.Date.Equal(existing.Date) {
		existing.Date = fetched.Date
		changed = true
	}
	if fetched.FeedTitle != "" && fetched.FeedTitle != existing.FeedTitle {
		existing.FeedTitle = fetched.FeedTitle
		changed = true
	}
	if fetched.FeedURL != "" && fetched.FeedURL != existing.FeedURL {
		existing.FeedURL = fetched.FeedURL
		changed = true
	}
	if !savedAt.IsZero() && !savedAt.Equal(existing.SavedAt) {
		existing.SavedAt = savedAt
		changed = true
	}
	if !existing.BodyHydrated {
		existing.BodyHydrated = true
		changed = true
	}
	return changed
}

// MarkRead marks an item as read. Returns true if it existed.
func (h *History) MarkRead(guid string) bool {
	item, ok := h.items[guid]
	if !ok || item == nil {
		return false
	}
	item.IsRead = true
	return true
}

// Item returns a history item by GUID.
func (h *History) Item(guid string) (*HistoryItem, bool) {
	item, ok := h.items[guid]
	return item, ok
}

// ToggleBookmark returns true if the item exists and the specific item's state was toggled.
func (h *History) ToggleBookmark(guid string) bool {
	item, ok := h.items[guid]
	if !ok || item == nil {
		return false
	}
	item.IsBookmarked = !item.IsBookmarked
	return true
}

// SetInsight sets AI-generated insight fields for an item.
func (h *History) SetInsight(guid, summary string, tags []string, updatedAt time.Time) bool {
	item, ok := h.items[guid]
	if !ok || item == nil {
		return false
	}
	item.AISummary = summary
	item.AITags = append(item.AITags[:0], tags...)
	item.AIUpdatedAt = updatedAt
	return true
}

// UpsertItem inserts or updates a history item by GUID.
func (h *History) UpsertItem(item *HistoryItem) {
	if h == nil || item == nil || strings.TrimSpace(item.GUID) == "" {
		return
	}
	if item.kind() == NewsDigestKind {
		item.BodyHydrated = true
	}
	h.items[item.GUID] = item
}

// BookmarkedItems returns all bookmarked items.
func (h *History) BookmarkedItems() []*HistoryItem {
	items := make([]*HistoryItem, 0)
	for _, hItem := range h.items {
		if hItem != nil && hItem.IsBookmarked {
			items = append(items, hItem)
		}
	}
	return items
}

// ItemsByFeed returns history items filtered by feed URL.
func (h *History) ItemsByFeed(feedURL string) []*HistoryItem {
	if feedURL == BookmarksURL {
		return h.BookmarkedItems()
	}

	items := make([]*HistoryItem, 0, len(h.items))
	for _, hItem := range h.items {
		if hItem == nil || hItem.kind() == NewsDigestKind {
			continue
		}
		if feedURL == AllFeedsURL || feedURL == NewsURL || hItem.FeedURL == feedURL {
			items = append(items, hItem)
		}
	}
	return items
}

// DigestItemsByDate returns all digest items for the specified date key.
func (h *History) DigestItemsByDate(dateKey string) []*HistoryItem {
	items := make([]*HistoryItem, 0)
	for _, hItem := range h.items {
		if hItem == nil || hItem.kind() != NewsDigestKind || digestDateKey(hItem, time.Local) != dateKey {
			continue
		}
		items = append(items, hItem)
	}
	sort.Slice(items, func(i, j int) bool {
		leftDate := historySortDate(items[i], time.Local)
		rightDate := historySortDate(items[j], time.Local)
		if !leftDate.Equal(rightDate) {
			if leftDate.IsZero() {
				return false
			}
			if rightDate.IsZero() {
				return true
			}
			return leftDate.After(rightDate)
		}
		return items[i].GUID < items[j].GUID
	})
	return items
}

// DigestItems returns all digest items sorted by digest_date (desc),
// then by article date (desc), then by GUID (asc).
func (h *History) DigestItems() []*HistoryItem {
	items := make([]*HistoryItem, 0)
	for _, hItem := range h.items {
		if hItem == nil || hItem.kind() != NewsDigestKind {
			continue
		}
		items = append(items, hItem)
	}
	sort.Slice(items, func(i, j int) bool {
		leftKey := digestDateKey(items[i], time.Local)
		rightKey := digestDateKey(items[j], time.Local)
		if leftKey == rightKey {
			leftDate := historySortDate(items[i], time.Local)
			rightDate := historySortDate(items[j], time.Local)
			if !leftDate.Equal(rightDate) {
				if leftDate.IsZero() {
					return false
				}
				if rightDate.IsZero() {
					return true
				}
				return leftDate.After(rightDate)
			}
			return items[i].GUID < items[j].GUID
		}
		if leftKey == "" {
			return false
		}
		if rightKey == "" {
			return true
		}
		return leftKey > rightKey
	})
	return items
}

// ReplaceDigestItemsByDate upserts digest items for the date while keeping
// previously generated items for the same date.
func (h *History) ReplaceDigestItemsByDate(dateKey string, items []*HistoryItem) {
	for _, item := range items {
		if item == nil || item.GUID == "" {
			continue
		}
		item.Kind = NewsDigestKind
		item.DigestDate = dateKey
		item.BodyHydrated = true
		h.items[item.GUID] = item
	}
}

// TodayArticleItems returns today's article items in reverse-chronological order.
func (h *History) TodayArticleItems(dateKey string, feeds []string, loc *time.Location) []*HistoryItem {
	allowedFeeds := make(map[string]struct{}, len(feeds))
	for _, feed := range feeds {
		if feed == "" {
			continue
		}
		allowedFeeds[feed] = struct{}{}
	}

	items := make([]*HistoryItem, 0)
	for _, hItem := range h.items {
		if hItem == nil || hItem.kind() == NewsDigestKind {
			continue
		}
		if len(allowedFeeds) > 0 {
			if _, ok := allowedFeeds[hItem.FeedURL]; !ok {
				continue
			}
		}
		if historyDateKey(hItem, loc) != dateKey {
			continue
		}
		items = append(items, hItem)
	}

	sort.Slice(items, func(i, j int) bool {
		return historySortDate(items[i], loc).After(historySortDate(items[j], loc))
	})
	return items
}

func historySortDate(item *HistoryItem, loc *time.Location) time.Time {
	if item == nil {
		return time.Time{}
	}
	date := item.Date
	if date.IsZero() {
		date = item.SavedAt
	}
	return inLocation(date, loc)
}

func historyDateKey(item *HistoryItem, loc *time.Location) string {
	date := historySortDate(item, loc)
	if date.IsZero() {
		return ""
	}
	return date.Format("2006-01-02")
}

func digestDateKey(item *HistoryItem, loc *time.Location) string {
	if item == nil {
		return ""
	}
	if key := strings.TrimSpace(item.DigestDate); key != "" {
		return key
	}
	return historyDateKey(item, loc)
}

func inLocation(date time.Time, loc *time.Location) time.Time {
	if date.IsZero() {
		return date
	}
	if loc == nil {
		loc = time.Local
	}
	return date.In(loc)
}

func (h *HistoryItem) kind() string {
	if h == nil || h.Kind == "" {
		return ArticleKind
	}
	return h.Kind
}

// RelatedItems resolves RelatedGUIDs to article items while preserving GUID order.
func (h *History) RelatedItems(digest *HistoryItem) []*HistoryItem {
	if digest == nil || len(digest.RelatedGUIDs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(digest.RelatedGUIDs))
	items := make([]*HistoryItem, 0, len(digest.RelatedGUIDs))
	for _, guid := range digest.RelatedGUIDs {
		guid = strings.TrimSpace(guid)
		if guid == "" {
			continue
		}
		if _, dup := seen[guid]; dup {
			continue
		}
		seen[guid] = struct{}{}
		item, ok := h.items[guid]
		if !ok || item == nil || item.kind() == NewsDigestKind {
			continue
		}
		items = append(items, item)
	}
	return items
}
