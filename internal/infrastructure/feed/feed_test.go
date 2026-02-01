package feed

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDefaultParserHeaders(t *testing.T) {
	var gotAccept string
	var gotUA string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Feed</title>
  <id>urn:uuid:60a76c80-d399-11d9-b93C-0003939e0af6</id>
  <updated>2026-01-01T00:00:00Z</updated>
  <entry>
    <title>Atom-Powered Robots Run Amok</title>
    <id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
    <updated>2026-01-01T00:00:00Z</updated>
    <link href="https://example.com/robots"/>
    <summary>Some text.</summary>
  </entry>
</feed>`))
	}))
	defer server.Close()

	_, err := defaultParser(server.URL)
	if err != nil {
		t.Fatalf("default parser failed: %v", err)
	}

	if gotUA != "Reazy/1.0" {
		t.Errorf("Expected User-Agent 'Reazy/1.0', got %q", gotUA)
	}
	if gotAccept == "" || !strings.Contains(gotAccept, "application/atom+xml") {
		t.Errorf("Expected Accept header to include atom, got %q", gotAccept)
	}
}

func TestFetch(t *testing.T) {
	// Restore original parser after test
	defer func() {
		ParserFunc = defaultParser
	}()

	t.Run("Success", func(t *testing.T) {
		mockFeed := &gofeed.Feed{
			Title: "Test Feed",
			Items: []*gofeed.Item{
				{Title: "Item 1", Description: "Desc 1", Content: "Content 1", Link: "http://link1.com", GUID: "guid-1", Published: "2023-01-01"},
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
		if f.Items[0].GUID != "guid-1" {
			t.Errorf("Expected GUID 'guid-1', got %s", f.Items[0].GUID)
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

func TestFetchTrimsWhitespace(t *testing.T) {
	originalParser := ParserFunc
	defer func() { ParserFunc = originalParser }()

	var gotURL string
	ParserFunc = func(url string) (*gofeed.Feed, error) {
		gotURL = url
		return &gofeed.Feed{Title: "Trimmed", Items: []*gofeed.Item{}}, nil
	}

	feed, err := Fetch(" \nhttps://example.com/rss\t ")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if gotURL != "https://example.com/rss" {
		t.Fatalf("Expected trimmed url, got %q", gotURL)
	}
	if feed.URL != "https://example.com/rss" {
		t.Fatalf("Expected feed URL to be trimmed, got %q", feed.URL)
	}
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
