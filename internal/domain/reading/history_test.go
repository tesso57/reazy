package reading

import (
	"testing"
	"time"
)

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
		"d1": {
			GUID:       "d1",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-14",
			FeedURL:    NewsURL,
		},
	}
	h := NewHistory(items)

	tests := []struct {
		name      string
		feedURL   string
		wantCount int
	}{
		{"specific feed", "url1", 2},
		{"all feeds", AllFeedsURL, 3},
		{"news", NewsURL, 3},
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

func TestHistory_DigestItemsAndReplace(t *testing.T) {
	h := NewHistory(map[string]*HistoryItem{
		"d_old_1": {
			GUID:       "d_old_1",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-14",
			Title:      "Old 1",
		},
		"d_old_2": {
			GUID:       "d_old_2",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-14",
			Title:      "Old 2",
		},
		"d_keep": {
			GUID:       "d_keep",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-13",
			Title:      "Keep",
		},
		"a1": {
			GUID:  "a1",
			Title: "Article 1",
		},
	})

	newDigest := []*HistoryItem{
		{
			GUID:       "d_new_1",
			Title:      "New 1",
			DigestDate: "ignored",
		},
		{
			GUID:       "d_new_2",
			Title:      "New 2",
			DigestDate: "ignored",
		},
	}
	h.ReplaceDigestItemsByDate("2026-02-14", newDigest)

	got := h.DigestItemsByDate("2026-02-14")
	if len(got) != 4 {
		t.Fatalf("DigestItemsByDate count = %d, want 4", len(got))
	}
	found := make(map[string]bool, len(got))
	for _, item := range got {
		if item == nil {
			continue
		}
		found[item.GUID] = true
		if item.Kind != NewsDigestKind {
			t.Fatalf("digest kind should be %q", NewsDigestKind)
		}
	}
	for _, guid := range []string{"d_old_1", "d_old_2", "d_new_1", "d_new_2"} {
		if !found[guid] {
			t.Fatalf("digest %q should exist after upsert", guid)
		}
	}

	if _, ok := h.Item("d_old_1"); !ok {
		t.Fatal("old digest should be kept")
	}
	if _, ok := h.Item("d_old_2"); !ok {
		t.Fatal("old digest should be kept")
	}
	if _, ok := h.Item("d_keep"); !ok {
		t.Fatal("digest from another date should be kept")
	}
	if _, ok := h.Item("a1"); !ok {
		t.Fatal("article should be kept")
	}
}

func TestHistory_DigestItems_SortedByDate(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	h := NewHistory(map[string]*HistoryItem{
		"d_unknown": {
			GUID: "d_unknown",
			Kind: NewsDigestKind,
		},
		"d_2026_02_13": {
			GUID:       "d_2026_02_13",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-13",
		},
		"d_2026_02_14_a": {
			GUID:       "d_2026_02_14_a",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-14",
		},
		"d_2026_02_14_b": {
			GUID:       "d_2026_02_14_b",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-14",
		},
		"d_fallback_saved_at": {
			GUID:    "d_fallback_saved_at",
			Kind:    NewsDigestKind,
			SavedAt: time.Date(2026, 2, 12, 8, 0, 0, 0, loc),
		},
		"a1": {
			GUID: "a1",
			Kind: ArticleKind,
		},
	})

	got := h.DigestItems()
	if len(got) != 5 {
		t.Fatalf("DigestItems count = %d, want 5", len(got))
	}
	if got[0].GUID != "d_2026_02_14_a" || got[1].GUID != "d_2026_02_14_b" {
		t.Fatalf("top items should be 2026-02-14 digests, got %#v %#v", got[0], got[1])
	}
	if got[2].GUID != "d_2026_02_13" {
		t.Fatalf("third item should be 2026-02-13 digest, got %#v", got[2])
	}
	if got[3].GUID != "d_fallback_saved_at" {
		t.Fatalf("fourth item should use SavedAt fallback date, got %#v", got[3])
	}
	if got[4].GUID != "d_unknown" {
		t.Fatalf("unknown-date digest should be last, got %#v", got[4])
	}
}

func TestHistory_TodayArticleItems(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	h := NewHistory(map[string]*HistoryItem{
		"published_today": {
			GUID:    "published_today",
			Kind:    ArticleKind,
			FeedURL: "feed1",
			Date:    time.Date(2026, 2, 14, 10, 0, 0, 0, loc),
			SavedAt: time.Date(2026, 2, 14, 11, 0, 0, 0, loc),
		},
		"saved_fallback_today": {
			GUID:    "saved_fallback_today",
			Kind:    ArticleKind,
			FeedURL: "feed2",
			Date:    time.Time{},
			SavedAt: time.Date(2026, 2, 14, 12, 0, 0, 0, loc),
		},
		"other_day": {
			GUID:    "other_day",
			Kind:    ArticleKind,
			FeedURL: "feed1",
			Date:    time.Date(2026, 2, 13, 23, 59, 0, 0, loc),
			SavedAt: time.Date(2026, 2, 14, 0, 1, 0, 0, loc),
		},
		"digest_today": {
			GUID:       "digest_today",
			Kind:       NewsDigestKind,
			DigestDate: "2026-02-14",
			FeedURL:    NewsURL,
		},
		"other_feed": {
			GUID:    "other_feed",
			Kind:    ArticleKind,
			FeedURL: "feed3",
			Date:    time.Date(2026, 2, 14, 13, 0, 0, 0, loc),
		},
	})

	got := h.TodayArticleItems("2026-02-14", []string{"feed1", "feed2"}, loc)
	if len(got) != 2 {
		t.Fatalf("TodayArticleItems count = %d, want 2", len(got))
	}
	if got[0].GUID != "saved_fallback_today" {
		t.Fatalf("first item = %s, want saved_fallback_today", got[0].GUID)
	}
	if got[1].GUID != "published_today" {
		t.Fatalf("second item = %s, want published_today", got[1].GUID)
	}
}

func TestHistory_RelatedItems(t *testing.T) {
	h := NewHistory(map[string]*HistoryItem{
		"a1": {
			GUID:    "a1",
			Kind:    ArticleKind,
			FeedURL: "feed1",
		},
		"a2": {
			GUID:    "a2",
			Kind:    ArticleKind,
			FeedURL: "feed2",
		},
		"d1": {
			GUID: "d1",
			Kind: NewsDigestKind,
		},
	})

	digest := &HistoryItem{
		GUID:         "digest",
		Kind:         NewsDigestKind,
		RelatedGUIDs: []string{"a2", "d1", "missing", "a2", "a1"},
	}

	got := h.RelatedItems(digest)
	if len(got) != 2 {
		t.Fatalf("RelatedItems count = %d, want 2", len(got))
	}
	if got[0].GUID != "a2" || got[1].GUID != "a1" {
		t.Fatalf("unexpected related order: %#v", got)
	}
}

func TestHistory_SetInsight(t *testing.T) {
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	h := NewHistory(map[string]*HistoryItem{
		"1": {GUID: "1"},
	})

	ok := h.SetInsight("1", "short summary", []string{"go", "rss"}, now)
	if !ok {
		t.Fatal("SetInsight should return true for existing item")
	}

	item, exists := h.Item("1")
	if !exists {
		t.Fatal("Item should exist")
	}
	if item.AISummary != "short summary" {
		t.Fatalf("AISummary = %q, want %q", item.AISummary, "short summary")
	}
	if len(item.AITags) != 2 || item.AITags[0] != "go" || item.AITags[1] != "rss" {
		t.Fatalf("AITags = %#v, want [go rss]", item.AITags)
	}
	if !item.AIUpdatedAt.Equal(now) {
		t.Fatalf("AIUpdatedAt = %v, want %v", item.AIUpdatedAt, now)
	}

	if h.SetInsight("missing", "x", nil, now) {
		t.Fatal("SetInsight should return false for missing item")
	}
}
