package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/domain/reading"
	"github.com/tesso57/reazy/internal/infrastructure/history"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
	"github.com/tesso57/reazy/internal/presentation/tui/update"
)

func TestNewModel(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{"http://example.com/rss"},
		KeyMap: settings.KeyMapConfig{
			Up: "k",
		},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	if m.state.Session != state.FeedView {
		t.Error("Expected initial state to be feedView")
	}
	if m.state.Session != state.FeedView {
		t.Error("Expected initial state to be feedView")
	}
	if len(m.state.FeedList.Items()) != 3 { // All + Bookmarks + 1 Feed
		t.Errorf("Expected 3 feed items (All+Bookmarks+1), got %d", len(m.state.FeedList.Items()))
	}
}

func TestItemMethods(t *testing.T) {
	i := presenter.Item{
		TitleText:     "Title",
		Desc:          "Desc",
		Link:          "Link",
		Published:     "2023",
		Read:          true,
		FeedTitleText: "Feed",
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

	i2 := presenter.Item{Desc: "Desc"}
	if i2.Description() != "Desc" {
		t.Errorf("Description mismatch for empty published")
	}
}

func TestKeyMap(t *testing.T) {
	cfg := settings.KeyMapConfig{Up: "k"}
	km := state.NewKeyMap(cfg)

	if len(km.ShortHelp()) == 0 {
		t.Error("ShortHelp empty")
	}
	if len(km.FullHelp()) == 0 {
		t.Error("FullHelp empty")
	}
}

func TestUpdate(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{"http://example.com"},
		KeyMap: settings.KeyMapConfig{
			Up: "k", Down: "j", Left: "h", Right: "l",
			AddFeed: "a", Quit: "q",
			DeleteFeed: "x",
			Refresh:    "r", Open: "enter", Back: "esc",
			UpPage: "pgup", DownPage: "pgdn",
		},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	// Test Resize
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = tm.(*Model)
	if m.state.Width != 100 {
		t.Error("Resize failed")
	}

	// Test Key Quit - Now brings up dialog
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.state.Session != state.QuitView {
		t.Error("Expected quitView state after q")
	}
	// Cancel quit to return to feedView for subsequent tests
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.state.Session != state.FeedView {
		t.Error("Expected feedView state after n")
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m2 := tm.(*Model)
	if !m2.state.Help.ShowAll {
		t.Error("Help toggle failed")
	}
	// Toggle back off to ensure View tests render main content
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = tm.(*Model)
	if m.state.Help.ShowAll {
		t.Error("Help toggle off failed")
	}

	// Test Add Feed Flow
	// 1. Enter Add Mode
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = tm.(*Model)
	if m.state.Session != state.AddingFeedView {
		t.Error("Failed to switch to addingFeedView")
	}

	// 2. Type URL
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = tm.(*Model)
	if m.state.TextInput.Value() != "h" {
		t.Error("TextInput failed")
	}

	// 3. Cancel (Esc)
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state.Session != state.FeedView {
		t.Error("Esc failed to return to feedView")
	}

	// Test Submit Feed (Enter)
	m.state.Session = state.AddingFeedView
	m.state.TextInput.SetValue("http://test.com")
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tm.(*Model)
	if m.state.Session != state.FeedView {
		t.Error("Enter failed to return to feedView")
	}

	// Test Navigation (Feed -> Article)
	m.state.FeedList.SetItems([]list.Item{&presenter.Item{TitleText: "1. http://example.com", Link: "http://example.com"}})

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)
	if !m.state.Loading {
		t.Error("Expected loading state on feed select")
	}

	// Test Delete Feed
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Test Loading View
	m.state.Loading = true
	if len(m.View()) == 0 {
		t.Error("Loading view empty")
	}
	m.state.Loading = false

	// Test Open Browser
	oldOpen := OSOpenCmd
	defer func() { OSOpenCmd = oldOpen }()

	OSOpenCmd = func(_ string) *exec.Cmd {
		return exec.Command("echo", "mock open")
	}

	m.state.FeedList.SetItems([]list.Item{&presenter.Item{TitleText: "http://example.com/rss", Link: "http://example.com/article"}})
	m.state.Session = state.ArticleView
	m.state.ArticleList.SetItems([]list.Item{&presenter.Item{TitleText: "Article", Link: "http://example.com/article", FeedTitleText: "Test"}})

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Test Refresh
	m.state.Session = state.ArticleView // Reset state for refresh test
	m.state.CurrentFeed = &reading.Feed{URL: "http://example.com", Title: "Test"}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("Refresh expected cmd")
	}

	// Test Back Navigation
	m.state.Session = state.ArticleView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if tm.(*Model).state.Session != state.FeedView {
		t.Error("Back (Esc) failed")
	}

	// Test Error Msg
	m.state.Session = state.FeedView
	err := fmt.Errorf("fetch error")
	// Ensure selection is known link
	m.state.FeedList.SetItems([]list.Item{&presenter.Item{TitleText: "Err Feed", Link: "http://error.com"}})
	m.state.FeedList.Select(0)
	tm, _ = m.Update(update.FeedFetchedMsg{Err: err, URL: "http://error.com"})
	m = tm.(*Model)
	if m.state.Err != err {
		t.Error("Error not set")
	}
	if m.state.Loading {
		t.Error("Loading not cleared on error")
	}

	// Test Detail View Navigation
	m.state.Session = state.ArticleView
	m.state.ArticleList.SetItems([]list.Item{&presenter.Item{TitleText: "A", Desc: "Desc", AISummary: "Generated summary", Link: "L"}})
	// Enter -> Detail View
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // 'l' triggers detail view now
	m = tm.(*Model)
	if m.state.Session != state.DetailView {
		t.Error("Failed to enter detailView")
	}
	if !strings.Contains(m.state.Viewport.View(), "AI Summary") {
		t.Error("Viewport missing AI Summary section")
	}
	if !strings.Contains(m.state.Viewport.View(), "Article Body") {
		t.Error("Viewport missing Article Body section")
	}
	if !strings.Contains(m.state.Viewport.View(), "Generated summary") {
		t.Error("Viewport missing summary content")
	}
	// Esc -> Article View
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state.Session != state.ArticleView {
		t.Error("Failed to exit detailView")
	}

	// Test View/Init
	viewOutput := m.View()
	if len(viewOutput) == 0 {
		t.Error("View empty")
	}
	// Verify Header Layout Fix
	// Ensure Feed Title label is present if we select an item
	if m.state.Session == state.ArticleView && len(m.state.ArticleList.Items()) > 0 {
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

	m.state.Session = state.AddingFeedView
	if len(m.View()) == 0 {
		t.Error("Adding view empty")
	}
	// Test Pagination Keys
	// Create enough items to paginate
	pagItems := make([]list.Item, 100)
	for i := range pagItems {
		pagItems[i] = &presenter.Item{TitleText: fmt.Sprintf("Item %d", i), Link: "L", FeedTitleText: "Test Feed"}
	}
	m.state.ArticleList.SetItems(pagItems)
	m.state.ArticleList.SetHeight(10) // Small height to force pagination
	m.state.Session = state.ArticleView

	// Force selection to index 0 for View test
	m.state.ArticleList.Select(0)

	// Test View/Init
	viewOutput = m.View()
	if len(viewOutput) == 0 {
		t.Error("View empty")
	}
	// Verify Header Layout Fix
	// Ensure Feed Title label is present if we select an item
	if m.state.Session == state.ArticleView && len(m.state.ArticleList.Items()) > 0 {
		sel := m.state.ArticleList.SelectedItem().(*presenter.Item)
		if !strings.Contains(viewOutput, "🏷️") {
			t.Errorf("Expected Feed Title label (🏷️) in header. Selected item: %+v", sel)
		}
		// truncatedLink := sel.link
		// if len(truncatedLink) > m.width-4 {
		// 	truncatedLink = truncatedLink[:m.width-4] + "..."
		// }
	}
	if m.state.ArticleList.Paginator.Page != 0 {
		t.Error("Expected initial page 0")
	}

	// Press DownPage
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = tm.(*Model)
	if m.state.ArticleList.Paginator.Page == 0 {
		t.Error("Expected page to increase after PgDn")
	}

	// Press UpPage
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = tm.(*Model)
	if m.state.ArticleList.Paginator.Page != 0 {
		t.Error("Expected page to return to 0 after PgUp")
	}

	// Test AllFeeds Fetch Msg
	m.state.CurrentFeed = &reading.Feed{URL: reading.AllFeedsURL, Title: "All"}
	itemWithFeed := reading.Item{Title: "Title", FeedTitle: "TechCrunch", Link: "L"}
	msgAll := update.FeedFetchedMsg{
		Feed: &reading.Feed{
			URL:   reading.AllFeedsURL,
			Title: "All",
			Items: []reading.Item{itemWithFeed},
		},
		URL: reading.AllFeedsURL,
	}
	// Set selection to All Feeds for UI update
	m.state.FeedList.SetItems([]list.Item{&presenter.Item{TitleText: "All", Link: reading.AllFeedsURL}})
	m.state.FeedList.Select(0)

	update.HandleFeedFetchedMsg(m.state, msgAll, m.deps())
	// Check if title contains [TechCrunch]
	if len(m.state.ArticleList.Items()) > 0 {
		itm := m.state.ArticleList.Items()[0].(*presenter.Item)
		if !strings.Contains(itm.TitleText, "[TechCrunch]") {
			t.Error("All feeds title missing feed label")
		}
	} else {
		t.Error("All feeds items empty")
	}

	// Test Feed Switching Resets Cursor
	// 1. Scroll down in current feed
	m.state.Session = state.ArticleView
	m.state.ArticleList.Select(10) // Select an index > 0
	if m.state.ArticleList.Index() != 10 {
		// Just ensure it's not 0 for the test, although SetItems resets if we don't manage it?
		// Wait, simulated items are only 1 (All Feed above). We need more items.
		// Re-use pagItems
		m.state.ArticleList.SetItems(pagItems)
		m.state.ArticleList.Select(10)
	}

	// 2. Switch back to feed list
	m.state.Session = state.FeedView

	// 3. Select a feed (simulate 'l' or 'enter')
	// This triggers handleFeedViewKeys which should reset selection
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)

	// 4. Verify cursor is reset
	if m.state.ArticleList.Index() != 0 {
		t.Errorf("Expected cursor reset to 0, got %d", m.state.ArticleList.Index())
	}
	// Verify filter is reset (mock filter state if possible, but list.Model internals hard to set without interaction)
}

func TestDetailViewHeaderVisibleWithAISummary(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{"http://example.com"},
		KeyMap: settings.KeyMapConfig{
			Open:  "enter",
			Right: "l",
		},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	tm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = tm.(*Model)

	m.state.Session = state.ArticleView
	m.state.ArticleList.SetItems([]list.Item{
		&presenter.Item{
			TitleText:     "1. Header check",
			RawTitle:      "Header check",
			Desc:          "desc",
			Content:       "body",
			Link:          "https://example.com/post",
			FeedTitleText: "Example Feed",
			GUID:          "guid-1",
			AISummary:     "AI summary exists",
		},
	})
	m.state.ArticleList.Select(0)

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)
	if m.state.Session != state.DetailView {
		t.Fatalf("session = %v, want detail view", m.state.Session)
	}

	viewOutput := m.View()
	if !strings.Contains(viewOutput, "🔗") {
		t.Fatalf("expected link header in detail view, got: %s", viewOutput)
	}
	if !strings.Contains(viewOutput, "🏷️") {
		t.Fatalf("expected feed header in detail view, got: %s", viewOutput)
	}

	m.state.AIStatus = "AI: updated 2026-02-08 10:00"
	update.UpdateListSizes(m.state)
	viewOutput = m.View()
	if !strings.Contains(viewOutput, "AI: updated 2026-02-08 10:00") {
		t.Fatalf("expected AI status in footer, got: %s", viewOutput)
	}
	if strings.Contains(viewOutput, "AI: updated 2026-02-08 10:00\n\n") {
		t.Fatalf("AI status should not be inserted into main body, got: %s", viewOutput)
	}
	if !strings.Contains(viewOutput, "Reazy Feeds") {
		t.Fatalf("expected sidebar title to remain visible, got: %s", viewOutput)
	}
	if got, wantMax := lipgloss.Height(viewOutput), m.state.Height; got > wantMax {
		t.Fatalf("view height overflow in detail view with AI status: got=%d max=%d", got, wantMax)
	}
	for _, line := range strings.Split(viewOutput, "\n") {
		if w := lipgloss.Width(line); w > m.state.Width {
			t.Fatalf("view width overflow in detail view with AI status: got=%d max=%d line=%q", w, m.state.Width, line)
		}
	}
}

func TestArticleViewHeaderAndSidebarVisibleWithAIStatusFooter(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{"http://example.com"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	tm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = tm.(*Model)

	m.state.Session = state.ArticleView
	m.state.AIStatus = "AI: updated 2026-02-09 10:00"
	m.state.ArticleList.SetItems([]list.Item{
		&presenter.Item{
			TitleText:     "1. Header check",
			RawTitle:      "Header check",
			Desc:          "desc",
			Content:       "body",
			Link:          "https://example.com/post",
			FeedTitleText: "Example Feed",
			GUID:          "guid-2",
		},
	})
	m.state.ArticleList.Select(0)
	update.UpdateListSizes(m.state)

	viewOutput := m.View()
	if !strings.Contains(viewOutput, "Reazy Feeds") {
		t.Fatalf("expected sidebar title in article view, got: %s", viewOutput)
	}
	if !strings.Contains(viewOutput, "🔗") {
		t.Fatalf("expected link header in article view, got: %s", viewOutput)
	}
	if !strings.Contains(viewOutput, "🏷️") {
		t.Fatalf("expected feed header in article view, got: %s", viewOutput)
	}
	if !strings.Contains(viewOutput, "AI: updated 2026-02-09 10:00") {
		t.Fatalf("expected AI status in footer in article view, got: %s", viewOutput)
	}
	if strings.Contains(viewOutput, "AI: updated 2026-02-09 10:00\n\n") {
		t.Fatalf("AI status should not be inserted into article body, got: %s", viewOutput)
	}
	if got, wantMax := lipgloss.Height(viewOutput), m.state.Height; got > wantMax {
		t.Fatalf("view height overflow in article view with AI status: got=%d max=%d", got, wantMax)
	}
	for _, line := range strings.Split(viewOutput, "\n") {
		if w := lipgloss.Width(line); w > m.state.Width {
			t.Fatalf("view width overflow in article view with AI status: got=%d max=%d line=%q", w, m.state.Width, line)
		}
	}
}

func TestSidebarTitleVisibleDuringFilterInput(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{
			"https://example.com/feed1.xml",
			"https://example.com/feed2.xml",
			"https://example.com/feed3.xml",
		},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	tm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = tm.(*Model)

	assertStable := func(step string) {
		t.Helper()
		viewOutput := m.View()
		if !strings.Contains(viewOutput, "Reazy Feeds") {
			t.Fatalf("sidebar title disappeared at %s: %s", step, viewOutput)
		}
		if got, max := lipgloss.Height(viewOutput), m.state.Height; got > max {
			t.Fatalf("view height overflow at %s: got=%d max=%d", step, got, max)
		}
	}

	assertStable("initial")

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = tm.(*Model)
	assertStable("filter-start")

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = tm.(*Model)
	assertStable("filter-input-1")

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = tm.(*Model)
	assertStable("filter-input-2")
}

func TestFetchFeedCmd(t *testing.T) {
	cfg := settings.Settings{Feeds: []string{"http://example.com"}}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	// Invoke cmd
	cmd := update.FetchFeedCmd(m.reading, "http://example.com", m.state.Feeds)
	if cmd == nil {
		t.Error("fetchFeedCmd nil")
	}
	// Execute the command
	msg := cmd()
	if _, ok := msg.(update.FeedFetchedMsg); !ok {
		t.Error("Expected feedFetchedMsg")
	}

	// Test All Feeds Cmd
	cmd = update.FetchFeedCmd(m.reading, reading.AllFeedsURL, m.state.Feeds)
	if cmd == nil {
		t.Error("fetchFeedCmd (All) nil")
	}
	_ = cmd() // uses stub fetcher; no network calls.
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
	preItems := []*reading.HistoryItem{
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
	cfg := settings.Settings{
		Feeds:       []string{"http://example.com/rss"},
		HistoryFile: historyPath,
		KeyMap:      settings.KeyMapConfig{Right: "l"},
	}

	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, hm, stubFeedFetcher{})

	// Check if history loaded
	if len(m.state.History.Items()) != 2 {
		t.Errorf("Expected 2 history items, got %d", len(m.state.History.Items()))
	}

	// Simulate Fetch
	fetchedItems := []reading.Item{
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
	msg := update.FeedFetchedMsg{
		Feed: &reading.Feed{
			Title: "Test Feed",
			URL:   "http://example.com/rss",
			Items: fetchedItems,
		},
		URL: "http://example.com/rss",
	}

	// Ensure selection matches for UI update
	m.state.FeedList.SetItems([]list.Item{&presenter.Item{TitleText: "Test Feed", Link: "http://example.com/rss"}})
	m.state.FeedList.Select(0)

	update.HandleFeedFetchedMsg(m.state, msg, m.deps())

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

	items := m.state.ArticleList.Items()
	if len(items) != 3 {
		t.Errorf("Expected 3 items in list, got %d", len(items))
	}

	// Verify Merged Item IsRead
	// Find "Merged Title"
	foundMerged := false
	var feedTitleItem *presenter.Item
	for _, it := range items {
		i := it.(*presenter.Item)
		if strings.Contains(i.TitleText, "Merged Title") { // Title might have styling/number
			foundMerged = true
			if !i.Read {
				t.Error("Merged item should be Read")
			}
		}
		if strings.Contains(i.TitleText, "New Item") {
			if i.Read {
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
	m.state.ArticleList.Select(0)
	it := m.state.ArticleList.SelectedItem().(*presenter.Item)
	// Check feedTitle is populated
	if it.FeedTitleText == "" && feedTitleItem.FeedTitleText == "" {
		t.Log("FeedTitle is empty, check handleFeedFetchedMsg logic or test setup")
	}
	m.state.ArticleList.Select(0)
	it = m.state.ArticleList.SelectedItem().(*presenter.Item)
	if !strings.Contains(it.TitleText, "New Item") {
		t.Errorf("Expected first item to be New Item, got %s", it.TitleText)
	}

	// Trigger "Open" (l key)
	m.state.Session = state.ArticleView
	m.state.CurrentFeed = msg.Feed // Ensure current feed is set for context if needed
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)

	// Verify Read Status Updated in Model
	if !m.state.History.Items()["new-link"].IsRead {
		t.Error("New Item should be marked read in history map")
	}

	// Verify it enters detail view
	if m.state.Session != state.DetailView {
		t.Error("Should enter detail view")
	}

	// Wait for async history saves.
	time.Sleep(100 * time.Millisecond)
}
