package reading

import "testing"

func TestHistory_ToggleBookmark(t *testing.T) {
	tests := []struct {
		name         string
		initialItems map[string]*HistoryItem
		targetGUID   string
		wantResult   bool
		checkItem    func(*testing.T, *HistoryItem)
	}{
		{
			name:         "bookmark on non-existent item",
			initialItems: map[string]*HistoryItem{},
			targetGUID:   "unknown",
			wantResult:   false,
			checkItem:    nil,
		},
		{
			name: "toggle on (false -> true)",
			initialItems: map[string]*HistoryItem{
				"1": {GUID: "1", IsBookmarked: false},
			},
			targetGUID: "1",
			wantResult: true,
			checkItem: func(t *testing.T, item *HistoryItem) {
				if !item.IsBookmarked {
					t.Error("expected item to be bookmarked")
				}
			},
		},
		{
			name: "toggle off (true -> false)",
			initialItems: map[string]*HistoryItem{
				"1": {GUID: "1", IsBookmarked: true},
			},
			targetGUID: "1",
			wantResult: true,
			checkItem: func(t *testing.T, item *HistoryItem) {
				if item.IsBookmarked {
					t.Error("expected item to be unbookmarked")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHistory(tt.initialItems)
			got := h.ToggleBookmark(tt.targetGUID)

			if got != tt.wantResult {
				t.Errorf("ToggleBookmark() = %v, want %v", got, tt.wantResult)
			}

			if tt.checkItem != nil {
				item := h.items[tt.targetGUID]
				tt.checkItem(t, item)
			}
		})
	}
}

func TestHistory_ItemsByFeed(t *testing.T) {
	items := map[string]*HistoryItem{
		"1": {GUID: "1", FeedURL: "url1", IsBookmarked: true},
		"2": {GUID: "2", FeedURL: "url1", IsBookmarked: false},
		"3": {GUID: "3", FeedURL: "url2", IsBookmarked: true},
	}
	h := NewHistory(items)

	tests := []struct {
		name      string
		feedURL   string
		wantCount int
	}{
		{"specific feed", "url1", 2},
		{"all feeds", AllFeedsURL, 3},
		{"bookmarks", BookmarksURL, 2},
		{"unknown feed", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.ItemsByFeed(tt.feedURL)
			if len(got) != tt.wantCount {
				t.Errorf("ItemsByFeed(%q) count = %d, want %d", tt.feedURL, len(got), tt.wantCount)
			}
		})
	}
}
