package tui

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/application/usecase"
	"github.com/tesso57/reazy/internal/domain/reading"
	"github.com/tesso57/reazy/internal/domain/subscription"
	"github.com/tesso57/reazy/internal/infrastructure/history"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
	"github.com/tesso57/reazy/internal/presentation/tui/update"
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
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{})

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
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{})
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
	// Select first custom feed (0: All, 1: News, 2: Bookmarks, 3: first feed)
	m.state.FeedList.Select(3)
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
	if tm.(*Model).state.Session != state.FeedView {
		t.Error("Should stay in feedView when deleting All Feeds tab")
	}

	// Test 4: Try Delete News tab (Index 1)
	m.state.FeedList.Select(1)
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if tm.(*Model).state.Session != state.FeedView {
		t.Error("Should not allow deleting News tab")
	}

	// Test 5: Try Delete Bookmarks tab (Index 2)
	m.state.FeedList.Select(2)
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if tm.(*Model).state.Session != state.FeedView {
		t.Error("Should not allow deleting Bookmarks tab")
	}
}

func TestHandleFeedViewKeys_GroupFeeds(t *testing.T) {
	cfg := settings.Settings{
		Feeds:  []string{"https://news.ycombinator.com/rss", "https://github.com/golang/go/releases.atom", "https://planetpython.org/rss20.xml"},
		KeyMap: settings.KeyMapConfig{GroupFeeds: "z"},
	}
	repo := &stubSubscriptionRepo{feeds: cfg.Feeds}
	m := newTestModelWithInsightAndNewsDigestAndGroupingGenerator(
		cfg,
		repo,
		&stubHistoryRepo{},
		&stubFeedFetcher{},
		nil,
		nil,
		&stubFeedGroupingGenerator{
			groups: []subscription.FeedGroup{
				{
					Name:  "Tech",
					Feeds: []string{"https://news.ycombinator.com/rss", "https://github.com/golang/go/releases.atom"},
				},
			},
		},
	)
	m.state.Session = state.FeedView

	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = tm.(*Model)
	if !m.state.Loading {
		t.Fatal("Expected loading state while AI feed grouping runs")
	}
	if cmd == nil {
		t.Fatal("Expected feed grouping command")
	}

	tm, _ = m.Update(update.FeedGroupingCompletedMsg{
		Feeds: []string{
			"https://news.ycombinator.com/rss",
			"https://github.com/golang/go/releases.atom",
			"https://planetpython.org/rss20.xml",
		},
		Groups: []subscription.FeedGroup{
			{
				Name:  "Tech",
				Feeds: []string{"https://news.ycombinator.com/rss", "https://github.com/golang/go/releases.atom"},
			},
		},
		Ungrouped: []string{"https://planetpython.org/rss20.xml"},
	})
	m = tm.(*Model)

	if m.state.Loading {
		t.Fatal("Expected loading false after feed grouping result")
	}
	if len(m.state.FeedGroups) != 1 || m.state.FeedGroups[0].Name != "Tech" {
		t.Fatalf("unexpected feed groups: %#v", m.state.FeedGroups)
	}
	if !strings.Contains(m.state.StatusMessage, "AI grouped 2 feeds into 1 groups") {
		t.Fatalf("unexpected status message: %q", m.state.StatusMessage)
	}

	foundHeader := false
	for _, item := range m.state.FeedList.Items() {
		it, ok := item.(*presenter.Item)
		if !ok {
			continue
		}
		if it.IsSectionHeader() && it.RawTitle == "Tech" {
			foundHeader = true
			break
		}
	}
	if !foundHeader {
		t.Fatal("expected grouped sidebar header after AI grouping")
	}
}

func TestHandleFeedViewKeys_SummarizeTriggersGrouping(t *testing.T) {
	cfg := settings.Settings{
		Feeds:  []string{"https://news.ycombinator.com/rss", "https://planetpython.org/rss20.xml"},
		KeyMap: settings.KeyMapConfig{GroupFeeds: "z", Summarize: "s"},
	}
	repo := &stubSubscriptionRepo{feeds: cfg.Feeds}
	m := newTestModelWithInsightAndNewsDigestAndGroupingGenerator(
		cfg,
		repo,
		&stubHistoryRepo{},
		&stubFeedFetcher{},
		nil,
		nil,
		&stubFeedGroupingGenerator{
			groups: []subscription.FeedGroup{
				{
					Name:  "Tech",
					Feeds: []string{"https://news.ycombinator.com/rss"},
				},
			},
		},
	)
	m.state.Session = state.FeedView

	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = tm.(*Model)
	if !m.state.Loading {
		t.Fatal("Expected loading state while AI feed grouping runs via summarize key")
	}
	if cmd == nil {
		t.Fatal("Expected feed grouping command on summarize key in feed view")
	}
}

func TestHandleFeedViewKeys_GroupJumpByNumber(t *testing.T) {
	cfg := settings.Settings{
		FeedGroups: []subscription.FeedGroup{
			{
				Name:  "Tech",
				Feeds: []string{"https://a.example.com/rss", "https://b.example.com/rss"},
			},
			{
				Name:  "AI",
				Feeds: []string{"https://c.example.com/rss"},
			},
		},
		Feeds: []string{"https://d.example.com/rss"},
	}
	m := newTestModel(
		cfg,
		&stubSubscriptionRepo{feeds: cfg.Feeds, groups: cfg.FeedGroups},
		&stubHistoryRepo{},
		&stubFeedFetcher{},
	)
	m.state.Session = state.FeedView

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://a.example.com/rss" {
		t.Fatalf("key '1' should jump to first group feed, got %#v", m.state.FeedList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://c.example.com/rss" {
		t.Fatalf("key '2' should jump to second group feed, got %#v", m.state.FeedList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://d.example.com/rss" {
		t.Fatalf("key '3' should jump to ungrouped feed, got %#v", m.state.FeedList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://d.example.com/rss" {
		t.Fatalf("out-of-range group key should keep current selection, got %#v", m.state.FeedList.SelectedItem())
	}
}

func TestHandleFeedViewKeys_GroupJumpByJK(t *testing.T) {
	cfg := settings.Settings{
		FeedGroups: []subscription.FeedGroup{
			{
				Name:  "Tech",
				Feeds: []string{"https://a.example.com/rss"},
			},
			{
				Name:  "AI",
				Feeds: []string{"https://c.example.com/rss"},
			},
		},
		Feeds: []string{"https://d.example.com/rss"},
	}
	m := newTestModel(
		cfg,
		&stubSubscriptionRepo{feeds: cfg.Feeds, groups: cfg.FeedGroups},
		&stubHistoryRepo{},
		&stubFeedFetcher{},
	)
	m.state.Session = state.FeedView

	// From built-in tabs, J should jump to the first group.
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://a.example.com/rss" {
		t.Fatalf("key 'J' should jump to first group from built-ins, got %#v", m.state.FeedList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://c.example.com/rss" {
		t.Fatalf("second 'J' should jump to next group, got %#v", m.state.FeedList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://d.example.com/rss" {
		t.Fatalf("third 'J' should jump to ungrouped section, got %#v", m.state.FeedList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = tm.(*Model)
	if item, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok || item.Link != "https://c.example.com/rss" {
		t.Fatalf("key 'K' should jump to previous group, got %#v", m.state.FeedList.SelectedItem())
	}
}

func TestHandleArticleViewKeys_SectionJumpByNumber(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{"https://example.com/feed.xml"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{})
	m.state.Session = state.ArticleView
	m.state.ArticleList.SetItems([]list.Item{
		&presenter.Item{TitleText: "== 2026-02-15 (2) ==", SectionHeader: true},
		&presenter.Item{TitleText: "1. A", GUID: "a"},
		&presenter.Item{TitleText: "2. B", GUID: "b"},
		&presenter.Item{TitleText: "== 2026-02-14 (1) ==", SectionHeader: true},
		&presenter.Item{TitleText: "3. C", GUID: "c"},
		&presenter.Item{TitleText: "== 2026-02-13 (1) ==", SectionHeader: true},
		&presenter.Item{TitleText: "4. D", GUID: "d"},
	})
	m.state.ArticleList.Select(1)

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = tm.(*Model)
	if item, ok := m.state.ArticleList.SelectedItem().(*presenter.Item); !ok || item.GUID != "c" {
		t.Fatalf("key '2' should jump to second section, got %#v", m.state.ArticleList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = tm.(*Model)
	if item, ok := m.state.ArticleList.SelectedItem().(*presenter.Item); !ok || item.GUID != "d" {
		t.Fatalf("key '3' should jump to third section, got %#v", m.state.ArticleList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	m = tm.(*Model)
	if item, ok := m.state.ArticleList.SelectedItem().(*presenter.Item); !ok || item.GUID != "d" {
		t.Fatalf("out-of-range key should keep current selection, got %#v", m.state.ArticleList.SelectedItem())
	}
}

func TestHandleArticleViewKeys_SectionJumpByJK(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{"https://example.com/feed.xml"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{})
	m.state.Session = state.ArticleView
	m.state.ArticleList.SetItems([]list.Item{
		&presenter.Item{TitleText: "== 2026-02-15 (2) ==", SectionHeader: true},
		&presenter.Item{TitleText: "1. A", GUID: "a"},
		&presenter.Item{TitleText: "2. B", GUID: "b"},
		&presenter.Item{TitleText: "== 2026-02-14 (1) ==", SectionHeader: true},
		&presenter.Item{TitleText: "3. C", GUID: "c"},
		&presenter.Item{TitleText: "== 2026-02-13 (1) ==", SectionHeader: true},
		&presenter.Item{TitleText: "4. D", GUID: "d"},
	})
	m.state.ArticleList.Select(1)

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = tm.(*Model)
	if item, ok := m.state.ArticleList.SelectedItem().(*presenter.Item); !ok || item.GUID != "c" {
		t.Fatalf("key 'J' should jump to next section, got %#v", m.state.ArticleList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = tm.(*Model)
	if item, ok := m.state.ArticleList.SelectedItem().(*presenter.Item); !ok || item.GUID != "d" {
		t.Fatalf("second 'J' should jump to next section, got %#v", m.state.ArticleList.SelectedItem())
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = tm.(*Model)
	if item, ok := m.state.ArticleList.SelectedItem().(*presenter.Item); !ok || item.GUID != "c" {
		t.Fatalf("key 'K' should jump to previous section, got %#v", m.state.ArticleList.SelectedItem())
	}
}

func TestHandleFeedViewKeys_GroupFeedsError(t *testing.T) {
	cfg := settings.Settings{
		Feeds:  []string{"https://a.example.com/rss", "https://b.example.com/rss"},
		KeyMap: settings.KeyMapConfig{GroupFeeds: "z"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{})
	m.state.Session = state.FeedView

	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = tm.(*Model)
	if !m.state.Loading {
		t.Fatal("Expected loading state while AI feed grouping runs")
	}
	if cmd == nil {
		t.Fatal("Expected feed grouping command")
	}

	tm, _ = m.Update(update.FeedGroupingCompletedMsg{
		Err: errors.New("codex integration is disabled"),
	})
	m = tm.(*Model)
	if m.state.Err == nil {
		t.Fatal("expected error on feed grouping failure")
	}
	if !strings.Contains(m.state.StatusMessage, "AI feed grouping failed") {
		t.Fatalf("unexpected status message: %q", m.state.StatusMessage)
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
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, hm, &stubFeedFetcher{})

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

func TestHandleArticleViewKeys_OpenSectionHeaderDoesNothing(t *testing.T) {
	cfg := settings.Settings{
		Feeds:  []string{"http://example.com"},
		KeyMap: settings.KeyMapConfig{Right: "l", Bookmark: "b", Summarize: "s"},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{})

	m.state.Session = state.ArticleView
	m.state.ArticleList.SetItems([]list.Item{
		&presenter.Item{
			TitleText:     "== 2026-02-14 (2) ==",
			SectionHeader: true,
		},
		&presenter.Item{
			TitleText: "1. Headline",
			GUID:      "guid-1",
			Link:      "https://example.com/article",
		},
	})
	m.state.ArticleList.Select(0)

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = tm.(*Model)
	if m.state.Session != state.ArticleView {
		t.Fatalf("session = %v, want article view", m.state.Session)
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = tm.(*Model)
	if m.state.Session != state.ArticleView {
		t.Fatalf("session = %v, want article view after bookmark on section", m.state.Session)
	}

	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = tm.(*Model)
	if cmd != nil {
		t.Fatal("summarize command should be nil on section header")
	}
	if m.state.Loading {
		t.Fatal("loading should remain false on section header summarize")
	}
}

func TestUpdateAddingFeedView_Esc(t *testing.T) {
	cfg := settings.Settings{}
	m := newTestModel(cfg, &stubSubscriptionRepo{}, &stubHistoryRepo{}, &stubFeedFetcher{})
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
	m := newTestModel(cfg, &stubSubscriptionRepo{}, &stubHistoryRepo{}, &stubFeedFetcher{})
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

func TestHandleArticleViewKeys_Summarize(t *testing.T) {
	cfg := settings.Settings{
		Feeds:  []string{"http://example.com"},
		KeyMap: settings.KeyMapConfig{Summarize: "s", ToggleSummary: "S"},
	}
	m := newTestModelWithInsightGenerator(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{}, &stubInsightGenerator{
		insight: usecase.Insight{
			Summary: "AI generated summary",
			Tags:    []string{"go", "rss"},
		},
	})

	guid := "guid-1"
	m.state.History.Items()[guid] = &reading.HistoryItem{
		GUID:  guid,
		Title: "Article title",
	}
	m.state.Session = state.ArticleView
	m.state.ArticleList.SetItems([]list.Item{
		&presenter.Item{
			RawTitle:  "Article title",
			TitleText: "1. Article title",
			GUID:      guid,
			Content:   "Body text",
		},
	})
	m.state.ArticleList.Select(0)

	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = tm.(*Model)
	if !m.state.Loading {
		t.Fatal("Expected loading state while insight generation runs")
	}
	if cmd == nil {
		t.Fatal("Expected insight command")
	}

	tm, _ = m.Update(update.InsightGeneratedMsg{
		GUID: guid,
		Insight: usecase.Insight{
			Summary: "AI generated summary",
			Tags:    []string{"go", "rss"},
		},
	})
	m = tm.(*Model)

	if m.state.Loading {
		t.Fatal("Expected loading to stop after insight result")
	}
	item, _ := m.state.History.Item(guid)
	if item == nil || item.AISummary != "AI generated summary" {
		t.Fatalf("History insight not updated: %#v", item)
	}
	listItem := m.state.ArticleList.Items()[0].(*presenter.Item)
	if listItem.AISummary != "AI generated summary" {
		t.Fatalf("List item summary = %q, want AI summary", listItem.AISummary)
	}
}

func TestHandleDetailViewKeys_Summarize(t *testing.T) {
	cfg := settings.Settings{
		Feeds:  []string{"http://example.com"},
		KeyMap: settings.KeyMapConfig{Summarize: "s"},
	}
	m := newTestModelWithInsightGenerator(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, &stubFeedFetcher{}, &stubInsightGenerator{
		insight: usecase.Insight{
			Summary: "Detail AI summary",
			Tags:    []string{"tag1", "tag2"},
		},
	})

	guid := "guid-2"
	m.state.History.Items()[guid] = &reading.HistoryItem{
		GUID:  guid,
		Title: "Detail article",
	}
	m.state.Session = state.DetailView
	m.state.Viewport.Width = 80
	m.state.Viewport.Height = 20
	m.state.ArticleList.SetItems([]list.Item{
		&presenter.Item{
			RawTitle:  "Detail article",
			TitleText: "1. Detail article",
			GUID:      guid,
			Content:   "Body text",
		},
	})
	m.state.ArticleList.Select(0)

	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = tm.(*Model)
	if !m.state.Loading {
		t.Fatal("Expected loading state in detail view summarize")
	}

	tm, _ = m.Update(update.InsightGeneratedMsg{
		GUID: guid,
		Insight: usecase.Insight{
			Summary: "Detail AI summary",
			Tags:    []string{"tag1", "tag2"},
		},
	})
	m = tm.(*Model)
	if m.state.Loading {
		t.Fatal("Expected loading false after insight result")
	}
	if !strings.Contains(m.state.Viewport.View(), "AI Summary") {
		t.Fatalf("Viewport should include AI summary section, got: %s", m.state.Viewport.View())
	}
	if !strings.Contains(m.state.AIStatus, "AI: updated") {
		t.Fatalf("Expected AI status update message, got %q", m.state.AIStatus)
	}

	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = tm.(*Model)
	if !strings.Contains(m.state.Viewport.View(), "(hidden; press Shift+S to toggle)") {
		t.Fatalf("Viewport should indicate hidden summary, got: %s", m.state.Viewport.View())
	}
}
