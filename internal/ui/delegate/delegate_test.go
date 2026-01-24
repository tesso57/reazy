package delegate

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Mock items
type mockFeedItem struct {
	title string
	url   string
}

func (i mockFeedItem) FilterValue() string { return i.title }
func (i mockFeedItem) Title() string       { return i.title }
func (i mockFeedItem) URL() string         { return i.url }

type mockArticleItem struct {
	title     string
	isRead    bool
	feedTitle string
}

func (i mockArticleItem) FilterValue() string { return i.title }
func (i mockArticleItem) Title() string       { return i.title }
func (i mockArticleItem) IsRead() bool        { return i.isRead }
func (i mockArticleItem) FeedTitle() string   { return i.feedTitle }

func TestFeedDelegate_Render(t *testing.T) {
	d := NewFeedDelegate(lipgloss.Color("205"))
	buf := &bytes.Buffer{}
	l := list.New([]list.Item{}, d, 0, 0)

	item := mockFeedItem{title: "Test Feed", url: "http://test.com"}

	// Test Selected
	l.Select(0)
	d.Render(buf, l, 0, item)
	output := buf.String()
	if !strings.Contains(output, "Test Feed") {
		t.Errorf("Expected 'Test Feed' in output, got %q", output)
	}
	buf.Reset()

	// Test Unselected
	d.Render(buf, l, 1, item)
	output = buf.String()
	if !strings.Contains(output, "Test Feed") {
		t.Errorf("Expected 'Test Feed' in output, got %q", output)
	}
	// Note: Styles are hard to test exactly without checking ANSI codes,
	// but we verified content presence.
}

func TestArticleDelegate_Render(t *testing.T) {
	d := NewArticleDelegate()
	buf := &bytes.Buffer{}
	l := list.New([]list.Item{}, d, 0, 0)

	// Test Unread
	item := mockArticleItem{title: "Article 1", isRead: false}
	l.Select(0)
	d.Render(buf, l, 0, item)
	output := buf.String()
	if !strings.Contains(output, "Article 1") {
		t.Errorf("Expected 'Article 1', got %q", output)
	}
	buf.Reset()

	// Test Read (Faint)
	itemRead := mockArticleItem{title: "Article 2", isRead: true}
	d.Render(buf, l, 0, itemRead) // Selected
	output = buf.String()
	if !strings.Contains(output, "Article 2") {
		t.Errorf("Expected 'Article 2', got %q", output)
	}

	// Ensure Read item differs in output bytes due to formatting?
	// Lipgloss Faint adds ANSI codes.
	// But both selected and unselected add codes.
	// We just check content primarily for unit test.
}

func TestDelegates_Dimensions(t *testing.T) {
	fd := NewFeedDelegate(lipgloss.Color("205"))
	if fd.Height() != 1 {
		t.Error("FeedDelegate Height != 1")
	}
	if fd.Spacing() != 0 {
		t.Error("FeedDelegate Spacing != 0")
	}

	ad := NewArticleDelegate()
	if ad.Height() != 1 {
		t.Error("ArticleDelegate Height != 1")
	}
	if ad.Spacing() != 0 {
		t.Error("ArticleDelegate Spacing != 0")
	}
}
