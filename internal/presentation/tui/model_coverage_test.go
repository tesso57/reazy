package tui

import (
	"os/exec"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/domain/reading"
	"github.com/tesso57/reazy/internal/infrastructure/history"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
)

// mockFeedItem creates a simple item for testing
func mockFeedItem(title, link string) *presenter.Item {
	return &presenter.Item{
		TitleText:     title,
		Link:          link,
		FeedTitleText: "Test Feed",
	}
}

func TestHandleDetailViewKeys(t *testing.T) {
	cfg := settings.Settings{
		Feeds:       []string{"http://example.com"},
		HistoryFile: t.TempDir() + "/history.jsonl",
		KeyMap:      settings.KeyMapConfig{Right: "l", Left: "h", Open: "o"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	// Setup Detail View State
	m.state.Session = state.DetailView
	m.state.ArticleList.SetItems([]list.Item{mockFeedItem("Article 1", "http://example.com/1")})
	m.state.ArticleList.Select(0)

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
	if m.state.Session != state.ArticleView {
		t.Error("Expected to return to articleView on Left key")
	}

	// Test 3: Help Toggle
	m.state.Session = state.DetailView
	m.state.Help.ShowAll = false
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = tm.(*Model)
	if !m.state.Help.ShowAll {
		t.Error("Expected help to toggle on")
	}
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = tm.(*Model)
	if m.state.Help.ShowAll {
		t.Error("Expected help to toggle off")
	}
}

func TestHandleFeedViewKeys_AddDelete(t *testing.T) {
	cfg := settings.Settings{
		Feeds:  []string{"http://example.com/1", "http://example.com/2"},
		KeyMap: settings.KeyMapConfig{AddFeed: "a", DeleteFeed: "x"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})
	m.state.Session = state.FeedView

	// Test 1: Add Feed
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = tm.(*Model)
	if m.state.Session != state.AddingFeedView {
		t.Error("Expected addingFeedView state")
	}

	// Logic for adding feed view
	// Valid URL
	m.state.TextInput.SetValue("http://new.com")
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(*Model)
	if m.state.Session != state.FeedView {
		t.Error("Expected return to feedView after adding")
	}
	// Check if new was added (last item?)
	items := m.state.FeedList.Items()
	found := false
	for _, i := range items {
		if it, ok := i.(*presenter.Item); ok && it.Link == "http://new.com" {
			found = true
			break
		}
	}
	if !found {
		t.Error("New feed not found in list")
	}

	// Test 2: Delete Feed
	// Select index 1 (http://example.com/1, because 0 is All)
	m.state.FeedList.Select(1)
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = tm.(*Model)
	if m.state.Session != state.DeleteFeedView {
		t.Error("Should switch to DeleteFeedView on 'x'")
	}
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = tm.(*Model)

	// Should be removed
	items = m.state.FeedList.Items()
	for _, i := range items {
		if it, ok := i.(*presenter.Item); ok && it.Link == "http://example.com/1" {
			t.Error("Feed should be deleted")
		}
	}

	// Test 3: Try Delete All tab (Index 0)
	m.state.FeedList.Select(0)
	ct := len(m.state.FeedList.Items())
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if len(tm.(*Model).state.FeedList.Items()) != ct {
		t.Error("Should not allow deleting All Feeds tab")
	}
}

func TestHandleArticleViewKeys_MarkRead(t *testing.T) {
	// Setup History
	tmpDir := t.TempDir()
	histPath := tmpDir + "/hist.jsonl"
	hm := history.NewManager(histPath)

	cfg := settings.Settings{
		Feeds:       []string{"http://example.com"},
		HistoryFile: histPath,
		KeyMap:      settings.KeyMapConfig{Right: "l"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, hm, stubFeedFetcher{})

	guid := "test-guid"
	m.state.History.Items()[guid] = &reading.HistoryItem{GUID: guid, IsRead: false}

	m.state.Session = state.ArticleView
	it := mockFeedItem("Unread", "http://example.com/unread")
	it.GUID = guid
	it.Read = false

	m.state.ArticleList.SetItems([]list.Item{it})
	m.state.ArticleList.Select(0)

	// Action: Open article (Right) -> Should mark read
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)

	if m.state.Session != state.DetailView {
		t.Error("Expected switch to detailView")
	}

	// Check Model Item
	sel := m.state.ArticleList.Items()[0].(*presenter.Item)
	if !sel.Read {
		t.Error("Item in list should be marked read")
	}
	// Check History Map
	if !m.state.History.Items()[guid].IsRead {
		t.Error("History item should be marked read")
	}

	// Give background save goroutine time to release file handle
	// to prevent cleanup errors on Windows/Mac
	time.Sleep(100 * time.Millisecond)
}

func TestUpdateAddingFeedView_Esc(t *testing.T) {
	cfg := settings.Settings{}
	m := newTestModel(cfg, &stubSubscriptionRepo{}, &stubHistoryRepo{}, stubFeedFetcher{})
	m.state.Session = state.AddingFeedView

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state.Session != state.FeedView {
		t.Error("Expected feedView after Esc")
	}
}

func TestHandleArticleViewKeys_Refresh(t *testing.T) {
	cfg := settings.Settings{
		KeyMap: settings.KeyMapConfig{Refresh: "r"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{}, &stubHistoryRepo{}, stubFeedFetcher{})
	m.state.Session = state.ArticleView
	m.state.CurrentFeed = &reading.Feed{URL: "http://example.com"}

	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = tm.(*Model)
	if !m.state.Loading {
		t.Error("Expected loading state on refresh")
	}
	if cmd == nil {
		t.Error("Expected refresh command")
	}
}
