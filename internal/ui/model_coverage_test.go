package ui

import (
	"os/exec"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/config"
	"github.com/tesso57/reazy/internal/feed"
	"github.com/tesso57/reazy/internal/history"
)

// mockFeedItem creates a simple item for testing
func mockFeedItem(title, link string) *item {
	return &item{
		title:     title,
		link:      link,
		feedTitle: "Test Feed",
	}
}

func TestHandleDetailViewKeys(t *testing.T) {
	cfg := &config.Config{
		Feeds:       []string{"http://example.com"},
		HistoryFile: t.TempDir() + "/history.jsonl",
		KeyMap:      config.KeyMapConfig{Right: "l", Left: "h", Open: "o"},
	}
	m := NewModel(cfg)

	// Setup Detail View State
	m.state = detailView
	m.articleList.SetItems([]list.Item{mockFeedItem("Article 1", "http://example.com/1")})
	m.articleList.Select(0)

	// Test 1: Open Browser (Mocked)
	// We need to swap OSOpenCmd temporarily
	oldOpen := OSOpenCmd
	defer func() { OSOpenCmd = oldOpen }()

	openedURL := ""
	OSOpenCmd = func(url string) *exec.Cmd {
		openedURL = url
		return exec.Command("echo", "mock")
	}

	// Trigger Open
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if openedURL != "http://example.com/1" {
		t.Errorf("Expected to open url 'http://example.com/1', got '%s'", openedURL)
	}

	// Test 2: Back (Left) -> Article View
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = tm.(*Model)
	if m.state != articleView {
		t.Error("Expected to return to articleView on Left key")
	}

	// Test 3: Help Toggle
	m.state = detailView
	m.help.ShowAll = false
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = tm.(*Model)
	if !m.help.ShowAll {
		t.Error("Expected help to toggle on")
	}
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = tm.(*Model)
	if m.help.ShowAll {
		t.Error("Expected help to toggle off")
	}
}

func TestHandleFeedViewKeys_AddDelete(t *testing.T) {
	cfg := &config.Config{
		Feeds:  []string{"http://example.com/1", "http://example.com/2"},
		KeyMap: config.KeyMapConfig{AddFeed: "a", DeleteFeed: "x"},
	}
	m := NewModel(cfg)
	m.state = feedView

	// Test 1: Add Feed
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = tm.(*Model)
	if m.state != addingFeedView {
		t.Error("Expected addingFeedView state")
	}

	// Logic for adding feed view
	// Valid URL
	m.textInput.SetValue("http://new.com")
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(*Model)
	if m.state != feedView {
		t.Error("Expected return to feedView after adding")
	}
	// Check if new was added (last item?)
	items := m.feedList.Items()
	found := false
	for _, i := range items {
		if it, ok := i.(*item); ok && it.link == "http://new.com" {
			found = true
			break
		}
	}
	if !found {
		t.Error("New feed not found in list")
	}

	// Test 2: Delete Feed
	// Select index 1 (http://example.com/1, because 0 is All)
	m.feedList.Select(1)
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = tm.(*Model)

	// Should be removed
	items = m.feedList.Items()
	for _, i := range items {
		if it, ok := i.(*item); ok && it.link == "http://example.com/1" {
			t.Error("Feed should be deleted")
		}
	}

	// Test 3: Try Delete All tab (Index 0)
	m.feedList.Select(0)
	ct := len(m.feedList.Items())
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if len(tm.(*Model).feedList.Items()) != ct {
		t.Error("Should not allow deleting All Feeds tab")
	}
}

func TestHandleArticleViewKeys_MarkRead(t *testing.T) {
	// Setup History
	tmpDir := t.TempDir()
	histPath := tmpDir + "/hist.jsonl"
	hm := history.NewManager(histPath)

	cfg := &config.Config{
		Feeds:       []string{"http://example.com"},
		HistoryFile: histPath,
		KeyMap:      config.KeyMapConfig{Right: "l"},
	}
	m := NewModel(cfg)
	m.historyMgr = hm

	guid := "test-guid"
	m.history[guid] = &history.Item{GUID: guid, IsRead: false}

	m.state = articleView
	it := mockFeedItem("Unread", "http://example.com/unread")
	it.guid = guid
	it.isRead = false

	m.articleList.SetItems([]list.Item{it})
	m.articleList.Select(0)

	// Action: Open article (Right) -> Should mark read
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)

	if m.state != detailView {
		t.Error("Expected switch to detailView")
	}

	// Check Model Item
	sel := m.articleList.Items()[0].(*item)
	if !sel.isRead {
		t.Error("Item in list should be marked read")
	}
	// Check History Map
	if !m.history[guid].IsRead {
		t.Error("History item should be marked read")
	}

	// Give background save goroutine time to release file handle
	// to prevent cleanup errors on Windows/Mac
	time.Sleep(100 * time.Millisecond)
}

func TestUpdateAddingFeedView_Esc(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.state = addingFeedView

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state != feedView {
		t.Error("Expected feedView after Esc")
	}
}

func TestHandleArticleViewKeys_Refresh(t *testing.T) {
	cfg := &config.Config{
		KeyMap: config.KeyMapConfig{Refresh: "r"},
	}
	m := NewModel(cfg)
	m.state = articleView
	m.currentFeed = &feed.Feed{URL: "http://example.com"}

	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = tm.(*Model)
	if !m.loading {
		t.Error("Expected loading state on refresh")
	}
	if cmd == nil {
		t.Error("Expected refresh command")
	}
}
