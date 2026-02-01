package presenter

import (
	"strings"
	"testing"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

func TestBuildArticleListItems(t *testing.T) {
	now := time.Now()
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"guid1": {
			GUID:        "guid1",
			Title:       "Article One",
			Link:        "http://example.com/1",
			Date:        now,
			FeedURL:     "http://example.com/feed",
			FeedTitle:   "My Feed",
			Description: "Desc 1",
			Content:     "Content 1",
		},
		"guid2": {
			GUID:        "guid2",
			Title:       "Article Two",
			Link:        "http://example.com/2",
			Date:        now.Add(-1 * time.Hour),
			FeedURL:     "http://example.com/feed",
			FeedTitle:   "My Feed",
			Description: "Desc 2",
			Content:     "Content 2",
		},
	})

	// Test 1: Single Feed View
	items := BuildArticleListItems(history, "http://example.com/feed")
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Check numbering (1. Article One)
	// Sorted by date desc -> Article One is first (now), Article Two is second (now-1h)

	i1 := items[0].(*Item)
	if !strings.HasPrefix(i1.TitleText, "1. ") {
		t.Errorf("Expected first item to start with '1. ', got '%s'", i1.TitleText)
	}
	if !strings.Contains(i1.TitleText, "Article One") {
		t.Errorf("Expected first item to contain 'Article One', got '%s'", i1.TitleText)
	}

	i2 := items[1].(*Item)
	if !strings.HasPrefix(i2.TitleText, "2. ") {
		t.Errorf("Expected second item to start with '2. ', got '%s'", i2.TitleText)
	}

	// Test 2: All Feeds View ([Feed Name] Title)
	itemsAll := BuildArticleListItems(history, reading.AllFeedsURL)
	if len(itemsAll) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(itemsAll))
	}

	iA1 := itemsAll[0].(*Item)
	// Format: "1. [My Feed] Article One"
	if !strings.HasPrefix(iA1.TitleText, "1. [My Feed] Article One") {
		t.Errorf("Expected format '1. [My Feed] Article One', got '%s'", iA1.TitleText)
	}
}
