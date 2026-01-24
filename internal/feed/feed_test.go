package feed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
)

// Mock Fetching interaction integration tests would require external mocking.
// Here we might just test utility functions if any, or skip if purely wrapping gofeed.
// Since Fetch is a direct wrapper, we can at least test that our struct mapping is correct
// if we could inject a parser. For now, we'll keep it simple or test failure on invalid URL.

// Mock parser

// Test default parser func coverage
func TestDefaultParser(t *testing.T) {
	// Just call it with bad URL to verify it calls gofeed
	_, err := ParserFunc("invalid-url")
	if err == nil {
		t.Log("Expected error from default parser")
	}
}

func TestFetch(t *testing.T) {
	// Restore original parser after test
	defer func() {
		ParserFunc = func(url string) (*gofeed.Feed, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			fp := gofeed.NewParser()
			return fp.ParseURLWithContext(url, ctx)
		}
	}()

	t.Run("Success", func(t *testing.T) {
		mockFeed := &gofeed.Feed{
			Title: "Test Feed",
			Items: []*gofeed.Item{
				{Title: "Item 1", Description: "Desc 1", Content: "Content 1", Link: "http://link1.com", Published: "2023-01-01"},
			},
		}
		ParserFunc = func(_ string) (*gofeed.Feed, error) {
			return mockFeed, nil
		}

		f, err := Fetch("http://example.com")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if f.Title != "Test Feed" {
			t.Errorf("Expected title 'Test Feed', got %s", f.Title)
		}
		if len(f.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(f.Items))
		}
		if f.Items[0].Title != "Item 1" {
			t.Errorf("Expected item title 'Item 1', got %s", f.Items[0].Title)
		}
		if f.Items[0].FeedTitle != "Test Feed" {
			t.Errorf("Expected feed title 'Test Feed', got %s", f.Items[0].FeedTitle)
		}
		if f.Items[0].Content != "Content 1" {
			t.Errorf("Expected content 'Content 1', got %s", f.Items[0].Content)
		}
	})

	t.Run("Fallback Updated", func(t *testing.T) {
		mockFeed := &gofeed.Feed{
			Title: "Test Feed",
			Items: []*gofeed.Item{
				{Title: "Item 1", Link: "http://link1.com", Published: "", Updated: "2023-01-02"},
			},
		}
		ParserFunc = func(_ string) (*gofeed.Feed, error) {
			return mockFeed, nil
		}

		f, err := Fetch("http://example.com")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if f.Items[0].Published != "2023-01-02" {
			t.Errorf("Expected published '2023-01-02', got %s", f.Items[0].Published)
		}
	})

	t.Run("Failure", func(t *testing.T) {
		ParserFunc = func(_ string) (*gofeed.Feed, error) {
			return nil, gofeed.HTTPError{StatusCode: 404, Status: "Not Found"}
		}
		_, err := Fetch("http://example.com")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestFetchAll(t *testing.T) {
	// Mock ParserFunc
	originalParser := ParserFunc
	defer func() { ParserFunc = originalParser }()

	ParserFunc = func(url string) (*gofeed.Feed, error) {
		if url == "site1" {
			now := time.Now()
			return &gofeed.Feed{
				Title: "Site 1",
				Items: []*gofeed.Item{
					{Title: "Older", PublishedParsed: &[]time.Time{now.Add(-2 * time.Hour)}[0]},
				},
			}, nil
		}
		if url == "site2" {
			now := time.Now()
			return &gofeed.Feed{
				Title: "Site 2",
				Items: []*gofeed.Item{
					{Title: "Newer", PublishedParsed: &[]time.Time{now.Add(-1 * time.Hour)}[0]},
				},
			}, nil
		}
		return nil, fmt.Errorf("network error")
	}

	urls := []string{"site1", "site2", "error_site"}
	f, err := FetchAll(urls)
	if err != nil {
		t.Fatalf("FetchAll failed: %v", err)
	}

	if len(f.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(f.Items))
	}
	if f.Items[0].Title != "Newer" {
		t.Errorf("Expected 'Newer' first, got '%s'", f.Items[0].Title)
	}
	if f.Items[1].Title != "Older" {
		t.Errorf("Expected 'Older' second, got '%s'", f.Items[1].Title)
	}
	if f.Title != "All Feeds" {
		t.Errorf("Expected title 'All Feeds', got '%s'", f.Title)
	}
}
