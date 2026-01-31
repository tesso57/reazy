package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/config"
	"github.com/tesso57/reazy/internal/feed"
	"github.com/tesso57/reazy/internal/history"
)

func TestNewModel(t *testing.T) {
	cfg := &config.Config{
		Feeds: []string{"http://example.com/rss"},
		KeyMap: config.KeyMapConfig{
			Up: "k",
		},
	}
	m := NewModel(cfg)

	if m.state != feedView {
		t.Error("Expected initial state to be feedView")
	}
	if m.state != feedView {
		t.Error("Expected initial state to be feedView")
	}
	if len(m.feedList.Items()) != 2 { // All Tab + 1 Feed
		t.Errorf("Expected 2 feed items (All+1), got %d", len(m.feedList.Items()))
	}
}

func TestItemMethods(t *testing.T) {
	i := item{
		title:     "Title",
		desc:      "Desc",
		link:      "Link",
		published: "2023",
		isRead:    true,
		feedTitle: "Feed",
	}

	if i.FilterValue() != "Title" {
		t.Errorf("FilterValue mismatch")
	}
	if i.Title() != "Title" {
		t.Errorf("Title mismatch")
	}
	if i.URL() != "Link" {
		t.Errorf("URL mismatch")
	}
	if !i.IsRead() {
		t.Errorf("IsRead mismatch")
	}
	if i.FeedTitle() != "Feed" {
		t.Errorf("FeedTitle mismatch")
	}
	if i.Description() != "2023 - Desc" {
		t.Errorf("Description mismatch")
	}

	i2 := item{desc: "Desc"}
	if i2.Description() != "Desc" {
		t.Errorf("Description mismatch for empty published")
	}
}

func TestKeyMap(t *testing.T) {
	cfg := config.KeyMapConfig{Up: "k"}
	km := NewKeyMap(cfg)

	if len(km.ShortHelp()) == 0 {
		t.Error("ShortHelp empty")
	}
	if len(km.FullHelp()) == 0 {
		t.Error("FullHelp empty")
	}
}

func TestUpdate(t *testing.T) {
	cfg := &config.Config{
		Feeds: []string{"http://example.com"},
		KeyMap: config.KeyMapConfig{
			Up: "k", Down: "j", Left: "h", Right: "l",
			AddFeed: "a", Quit: "q",
			DeleteFeed: "x",
			Refresh:    "r", Open: "enter", Back: "esc",
			UpPage: "pgup", DownPage: "pgdn",
		},
	}
	m := NewModel(cfg)

	// Test Resize
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = tm.(*Model)
	if m.width != 100 {
		t.Error("Resize failed")
	}

	// Test Key Quit - Now brings up dialog
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.state != quitView {
		t.Error("Expected quitView state after q")
	}
	// Cancel quit to return to feedView for subsequent tests
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.state != feedView {
		t.Error("Expected feedView state after n")
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m2 := tm.(*Model)
	if !m2.help.ShowAll {
		t.Error("Help toggle failed")
	}
	// Toggle back off to ensure View tests render main content
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = tm.(*Model)
	if m.help.ShowAll {
		t.Error("Help toggle off failed")
	}

	// Test Add Feed Flow
	// 1. Enter Add Mode
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = tm.(*Model)
	if m.state != addingFeedView {
		t.Error("Failed to switch to addingFeedView")
	}

	// 2. Type URL
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = tm.(*Model)
	if m.textInput.Value() != "h" {
		t.Error("TextInput failed")
	}

	// 3. Cancel (Esc)
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state != feedView {
		t.Error("Esc failed to return to feedView")
	}

	// Test Submit Feed (Enter)
	m.state = addingFeedView
	m.textInput.SetValue("http://test.com")
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(*Model)
	if m.state != feedView {
		t.Error("Enter failed to return to feedView")
	}

	// Test Navigation (Feed -> Article)
	m.feedList.SetItems([]list.Item{&item{title: "1. http://example.com", link: "http://example.com"}})

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)
	if !m.loading {
		t.Error("Expected loading state on feed select")
	}

	// Test Delete Feed
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Test Loading View
	m.loading = true
	if len(m.View()) == 0 {
		t.Error("Loading view empty")
	}
	m.loading = false

	// Test Open Browser
	oldOpen := OSOpenCmd
	defer func() { OSOpenCmd = oldOpen }()

	OSOpenCmd = func(_ string) *exec.Cmd {
		return exec.Command("echo", "mock open")
	}

	m.feedList.SetItems([]list.Item{&item{title: "http://example.com/rss", link: "http://example.com/article"}})
	m.state = articleView
	m.articleList.SetItems([]list.Item{&item{title: "Article", link: "http://example.com/article", feedTitle: "Test"}})

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Test Refresh
	m.state = articleView // Reset state for refresh test
	m.currentFeed = &feed.Feed{URL: "http://example.com", Title: "Test"}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("Refresh expected cmd")
	}

	// Test Back Navigation
	m.state = articleView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if tm.(*Model).state != feedView {
		t.Error("Back (Esc) failed")
	}

	// Test Error Msg
	m.state = feedView
	err := fmt.Errorf("fetch error")
	// Ensure selection is known link
	m.feedList.SetItems([]list.Item{&item{title: "Err Feed", link: "http://error.com"}})
	m.feedList.Select(0)
	tm, _ = m.Update(feedFetchedMsg{err: err, url: "http://error.com"})
	m = tm.(*Model)
	if m.err != err {
		t.Error("Error not set")
	}
	if m.loading {
		t.Error("Loading not cleared on error")
	}

	// Test Detail View Navigation
	m.state = articleView
	m.articleList.SetItems([]list.Item{&item{title: "A", desc: "Desc", link: "L"}})
	// Enter -> Detail View
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // 'l' triggers detail view now
	m = tm.(*Model)
	if m.state != detailView {
		t.Error("Failed to enter detailView")
	}
	if !strings.Contains(m.viewport.View(), "Desc") {
		t.Error("Viewport missing content")
	}
	// Esc -> Article View
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state != articleView {
		t.Error("Failed to exit detailView")
	}

	// Test View/Init
	viewOutput := m.View()
	if len(viewOutput) == 0 {
		t.Error("View empty")
	}
	// Verify Header Layout Fix
	// Ensure Feed Title label is present if we select an item
	if m.state == articleView && len(m.articleList.Items()) > 0 {
		// sel := m.articleList.SelectedItem().(*item)
		if !strings.Contains(viewOutput, "🏷️") {
			// This might fail if the view doesn't render header for articleView?
			// Logic: header is rendered if m.articleList.SelectedItem() is ok.
			// It should be ok here.
			t.Error("Expected Feed Title label (🏷️) in header")
		}

	}
	if m.Init() == nil {
		t.Error("Init nil")
	}

	m.state = addingFeedView
	if len(m.View()) == 0 {
		t.Error("Adding view empty")
	}
	// Test Pagination Keys
	// Create enough items to paginate
	pagItems := make([]list.Item, 100)
	for i := range pagItems {
		pagItems[i] = &item{title: fmt.Sprintf("Item %d", i), link: "L", feedTitle: "Test Feed"}
	}
	m.articleList.SetItems(pagItems)
	m.articleList.SetHeight(10) // Small height to force pagination
	m.state = articleView

	// Force selection to index 0 for View test
	m.articleList.Select(0)

	// Test View/Init
	viewOutput = m.View()
	if len(viewOutput) == 0 {
		t.Error("View empty")
	}
	// Verify Header Layout Fix
	// Ensure Feed Title label is present if we select an item
	if m.state == articleView && len(m.articleList.Items()) > 0 {
		sel := m.articleList.SelectedItem().(*item)
		if !strings.Contains(viewOutput, "🏷️") {
			t.Errorf("Expected Feed Title label (🏷️) in header. Selected item: %+v", sel)
		}
		// truncatedLink := sel.link
		// if len(truncatedLink) > m.width-4 {
		// 	truncatedLink = truncatedLink[:m.width-4] + "..."
		// }
	}
	if m.articleList.Paginator.Page != 0 {
		t.Error("Expected initial page 0")
	}

	// Press DownPage
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = tm.(*Model)
	if m.articleList.Paginator.Page == 0 {
		t.Error("Expected page to increase after PgDn")
	}

	// Press UpPage
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = tm.(*Model)
	if m.articleList.Paginator.Page != 0 {
		t.Error("Expected page to return to 0 after PgUp")
	}

	// Test AllFeeds Fetch Msg
	m.currentFeed = &feed.Feed{URL: AllFeedsURL, Title: "All"}
	itemWithFeed := feed.Item{Title: "Title", FeedTitle: "TechCrunch", Link: "L"}
	msgAll := feedFetchedMsg{
		feed: &feed.Feed{
			URL:   AllFeedsURL,
			Title: "All",
			Items: []feed.Item{itemWithFeed},
		},
		url: AllFeedsURL,
	}
	// Set selection to All Feeds for UI update
	m.feedList.SetItems([]list.Item{&item{title: "All", link: AllFeedsURL}})
	m.feedList.Select(0)

	m.handleFeedFetchedMsg(msgAll)
	// Check if title contains [TechCrunch]
	if len(m.articleList.Items()) > 0 {
		itm := m.articleList.Items()[0].(*item)
		if !strings.Contains(itm.title, "[TechCrunch]") {
			t.Error("All feeds title missing feed label")
		}
	} else {
		t.Error("All feeds items empty")
	}

	// Test Feed Switching Resets Cursor
	// 1. Scroll down in current feed
	m.state = articleView
	m.articleList.Select(10) // Select an index > 0
	if m.articleList.Index() != 10 {
		// Just ensure it's not 0 for the test, although SetItems resets if we don't manage it?
		// Wait, simulated items are only 1 (All Feed above). We need more items.
		// Re-use pagItems
		m.articleList.SetItems(pagItems)
		m.articleList.Select(10)
	}

	// 2. Switch back to feed list
	m.state = feedView

	// 3. Select a feed (simulate 'l' or 'enter')
	// This triggers handleFeedViewKeys which should reset selection
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)

	// 4. Verify cursor is reset
	if m.articleList.Index() != 0 {
		t.Errorf("Expected cursor reset to 0, got %d", m.articleList.Index())
	}
	// Verify filter is reset (mock filter state if possible, but list.Model internals hard to set without interaction)
}

func TestFetchFeedCmd(t *testing.T) {
	// Invoke cmd
	cmd := fetchFeedCmd("http://example.com", nil)
	if cmd == nil {
		t.Error("fetchFeedCmd nil")
	}
	// Execute the command
	msg := cmd()
	if _, ok := msg.(feedFetchedMsg); !ok {
		t.Error("Expected feedFetchedMsg")
	}

	// Test All Feeds Cmd
	cmd = fetchFeedCmd(AllFeedsURL, []string{"http://example.com"})
	if cmd == nil {
		t.Error("fetchFeedCmd (All) nil")
	}
	_ = cmd() // fast enough with mock? internal/feed calls might block if real network.
	// We need to be careful if FetchAll makes network calls.
	// But in unit tests we ideally mock feed fetching.
	// The current codebase uses feed.Fetch/FetchAll which uses gofeed.
	// We might not want to actually run the cmd if it hits network.
	// But TestFetchFeedCmd was already running it.
	// Assuming feed.Fetch returns error or works.
}

func TestOpenBrowser(t *testing.T) {
	oldOpen := OSOpenCmd
	defer func() { OSOpenCmd = oldOpen }()

	called := false
	OSOpenCmd = func(_ string) *exec.Cmd {
		called = true
		return exec.Command("echo", "mock")
	}

	err := openBrowser("http://example.com")
	if err != nil {
		t.Errorf("openBrowser failed: %v", err)
	}
	if !called {
		t.Error("OSOpenCmd not called")
	}

	// Test unsupported platform (mocking nil return)
	OSOpenCmd = func(_ string) *exec.Cmd {
		return nil
	}
	err = openBrowser("http://example.com")
	if err == nil {
		t.Error("Expected error for unsupported platform")
	}
}

func TestHistoryIntegration(t *testing.T) {
	// Setup temp history file
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")

	// Pre-populate history
	hm := history.NewManager(historyPath)
	preItems := []*history.Item{
		{
			GUID:    "merged-link",
			Title:   "Merged Title",
			Link:    "merged-link",
			FeedURL: "http://example.com/rss",
			IsRead:  true,
			SavedAt: time.Now(),
			Date:    time.Now().Add(-1 * time.Hour),
		},
		{
			GUID:    "history-only",
			Title:   "History Only",
			Link:    "history-only",
			FeedURL: "http://example.com/rss",
			IsRead:  false,
			SavedAt: time.Now(),
			Date:    time.Now().Add(-2 * time.Hour),
		},
	}
	_ = hm.Save(preItems)

	// Setup Config
	cfg := &config.Config{
		Feeds:       []string{"http://example.com/rss"},
		HistoryFile: historyPath,
		KeyMap:      config.KeyMapConfig{Right: "l"},
	}

	m := NewModel(cfg)

	// Check if history loaded
	if len(m.history) != 2 {
		t.Errorf("Expected 2 history items, got %d", len(m.history))
	}

	// Simulate Fetch
	fetchedItems := []feed.Item{
		{
			Title:     "Merged Title",
			Link:      "merged-link",
			Date:      time.Now(),
			FeedURL:   "http://example.com/rss",
			FeedTitle: "Test Feed Title",
		},
		{
			Title:     "New Item",
			Link:      "new-link",
			Date:      time.Now(),
			FeedURL:   "http://example.com/rss",
			FeedTitle: "Test Feed Title",
		},
	}
	msg := feedFetchedMsg{
		feed: &feed.Feed{
			Title: "Test Feed",
			URL:   "http://example.com/rss",
			Items: fetchedItems,
		},
		url: "http://example.com/rss",
	}

	// Ensure selection matches for UI update
	m.feedList.SetItems([]list.Item{&item{title: "Test Feed", link: "http://example.com/rss"}})
	m.feedList.Select(0)

	m.handleFeedFetchedMsg(msg)

	// Verify Article List
	// Should contain 3 items: Merged, HistoryOnly, NewItem
	// Sorted by Date logic?
	// NewItem (Now), Merged (Now from fetch? Or history? We overwrite history date with fetch?)
	// Logic: if exists, we update missing FeedURL, but we didn't update Date.
	// But `Merged Title` in history has Date -1h. Fetch has Now.
	// `handleFeedFetchedMsg` logic:
	// if exist: update FeedURL if empty. Doesn't update Date.
	// So Merged will keep old Date (-1h).
	// NewItem: Now.
	// HistoryOnly: -2h.
	// Order: NewItem, Merged, HistoryOnly.

	items := m.articleList.Items()
	if len(items) != 3 {
		t.Errorf("Expected 3 items in list, got %d", len(items))
	}

	// Verify Merged Item IsRead
	// Find "Merged Title"
	foundMerged := false
	var feedTitleItem *item
	for _, it := range items {
		i := it.(*item)
		if strings.Contains(i.title, "Merged Title") { // Title might have styling/number
			foundMerged = true
			if !i.isRead {
				t.Error("Merged item should be Read")
			}
		}
		if strings.Contains(i.title, "New Item") {
			if i.isRead {
				t.Error("New item should be Unread")
			}
			// Use this item to test View feedTitle
			feedTitleItem = i
		}
	}
	if !foundMerged {
		t.Error("Merged item not found in list")
	}

	// Test Mark as Read
	// Select "New Item" (assuming it's at index 0 because it's newest)
	// Actually, confirm order first or find index.
	// NewItem is newest, so list[0] should be NewItem.
	m.articleList.Select(0)
	it := m.articleList.SelectedItem().(*item)
	// Check feedTitle is populated
	if it.feedTitle == "" && feedTitleItem.feedTitle == "" {
		t.Log("FeedTitle is empty, check handleFeedFetchedMsg logic or test setup")
	}
	m.articleList.Select(0)
	it = m.articleList.SelectedItem().(*item)
	if !strings.Contains(it.title, "New Item") {
		t.Errorf("Expected first item to be New Item, got %s", it.title)
	}

	// Trigger "Open" (l key)
	m.state = articleView
	m.currentFeed = msg.feed // Ensure current feed is set for context if needed
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)

	// Verify Read Status Updated in Model
	if !m.history["new-link"].IsRead {
		t.Error("New Item should be marked read in history map")
	}

	// Verify it enters detail view
	if m.state != detailView {
		t.Error("Should enter detail view")
	}
}
