package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tesso57/reazy/internal/config"
	"github.com/tesso57/reazy/internal/feed"
	"github.com/tesso57/reazy/internal/history"
	"github.com/tesso57/reazy/internal/ui/delegate"
	"github.com/tesso57/reazy/internal/ui/view"
)

type sessionState int

const (
	feedView sessionState = iota
	articleView
	detailView
	addingFeedView
	quitView
)

// AllFeedsURL is the special URL for the "All Feeds" view.
const AllFeedsURL = "internal://all"

// Model represents the main application state.
type Model struct {
	cfg           *config.Config
	state         sessionState
	feedList      list.Model
	articleList   list.Model
	textInput     textinput.Model
	viewport      viewport.Model
	help          help.Model
	spinner       spinner.Model
	loading       bool
	keys          KeyMap
	width         int
	height        int
	currentFeed   *feed.Feed
	err           error
	previousState sessionState

	historyMgr *history.Manager
	history    map[string]*history.Item // GUID -> Item
}

// KeyMap defines the keybindings for the application.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	UpPage     key.Binding
	DownPage   key.Binding
	Top        key.Binding
	Bottom     key.Binding
	Open       key.Binding
	Back       key.Binding
	Quit       key.Binding
	AddFeed    key.Binding
	DeleteFeed key.Binding
	Refresh    key.Binding
	Help       key.Binding
}

// ShortHelp returns a subset of keybindings for the help view.
func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Back, k.Open}
}

// FullHelp returns all keybindings for the help view.
func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Top, k.Bottom, k.UpPage, k.DownPage},
		{k.Open, k.Back, k.Quit},
		{k.AddFeed, k.DeleteFeed, k.Refresh, k.Help},
	}
}

// NewKeyMap creates a new KeyMap from the configuration.
func NewKeyMap(cfg config.KeyMapConfig) KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Up, ",")...),
			key.WithHelp(cfg.Up, "up"),
		),
		Down: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Down, ",")...),
			key.WithHelp(cfg.Down, "down"),
		),
		Left: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Left, ",")...),
			key.WithHelp(cfg.Left, "back/feeds"),
		),
		Right: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Right, ",")...),
			key.WithHelp(cfg.Right, "details"),
		),
		UpPage: key.NewBinding(
			key.WithKeys(strings.Split(cfg.UpPage, ",")...),
			key.WithHelp(cfg.UpPage, "pgup"),
		),
		DownPage: key.NewBinding(
			key.WithKeys(strings.Split(cfg.DownPage, ",")...),
			key.WithHelp(cfg.DownPage, "pgdn"),
		),
		Top: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Top, ",")...),
			key.WithHelp(cfg.Top, "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Bottom, ",")...),
			key.WithHelp(cfg.Bottom, "bottom"),
		),
		Open: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Open, ",")...),
			key.WithHelp(cfg.Open, "open"),
		),
		Back: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Back, ",")...),
			key.WithHelp(cfg.Back, "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Quit, ",")...),
			key.WithHelp(cfg.Quit, "quit"),
		),
		AddFeed: key.NewBinding(
			key.WithKeys(strings.Split(cfg.AddFeed, ",")...),
			key.WithHelp(cfg.AddFeed, "add"),
		),
		DeleteFeed: key.NewBinding(
			key.WithKeys(strings.Split(cfg.DeleteFeed, ",")...),
			key.WithHelp(cfg.DeleteFeed, "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys(strings.Split(cfg.Refresh, ",")...),
			key.WithHelp(cfg.Refresh, "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

// NewModel creates a new application model.
func NewModel(cfg *config.Config) *Model {
	l := list.New([]list.Item{}, delegate.NewFeedDelegate(lipgloss.Color(cfg.Theme.FeedName)), 0, 0)
	l.Title = "Reazy Feeds"
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()

	l.SetItems([]list.Item{})

	al := list.New([]list.Item{}, delegate.NewArticleDelegate(), 0, 0)
	al.Title = "Articles"
	al.SetShowTitle(false)
	al.SetShowHelp(false)
	al.DisableQuitKeybindings()

	ti := textinput.New()
	ti.Placeholder = "https://example.com/rss"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1)

	// Initialize History
	hm := history.NewManager(cfg.HistoryFile)
	hist, _ := hm.Load() // Ignore error on startup, just start fresh if fail
	if hist == nil {
		hist = make(map[string]*history.Item)
	}

	m := &Model{
		cfg:         cfg,
		state:       feedView,
		feedList:    l,
		articleList: al,
		textInput:   ti,
		viewport:    vp,
		help:        help.New(),
		spinner:     s,
		keys:        NewKeyMap(cfg.KeyMap),
		historyMgr:  hm,
		history:     hist,
	}
	// Inject pagination keys
	l.KeyMap.PrevPage = m.keys.UpPage
	l.KeyMap.NextPage = m.keys.DownPage
	al.KeyMap.PrevPage = m.keys.UpPage
	al.KeyMap.NextPage = m.keys.DownPage

	m.refreshFeedList()
	m.refreshFeedList()

	// Initial population of article list for "All Feeds" (default selection)
	m.updateArticleList(AllFeedsURL)

	return m
}

func (m *Model) refreshFeedList() {
	items := make([]list.Item, len(m.cfg.Feeds)+1)
	// Add "All" tab at the top
	items[0] = &item{title: "0. * All Feeds", link: AllFeedsURL}
	for i, f := range m.cfg.Feeds {
		// Display number + url + icon
		// e.g. "1.  https://example.com"
		// e.g. "1. https://example.com"
		items[i+1] = &item{title: fmt.Sprintf("%d. %s", i+1, f), link: f}
	}
	m.feedList.SetItems(items)
}

type item struct {
	title     string
	desc      string
	content   string
	link      string
	published string
	guid      string
	isRead    bool
	feedTitle string
	feedURL   string // Added specifically for grouping logic if needed in delegate
}

func (i *item) FilterValue() string { return i.title }
func (i *item) Title() string       { return i.title }
func (i *item) URL() string         { return i.link }
func (i *item) IsRead() bool        { return i.isRead }
func (i *item) FeedTitle() string   { return i.feedTitle }
func (i *item) Description() string {
	if i.published != "" {
		return fmt.Sprintf("%s - %s", i.published, i.desc)
	}
	return i.desc
}

type feedFetchedMsg struct {
	feed *feed.Feed
	err  error
	url  string
}

func fetchFeedCmd(url string, allFeeds []string) tea.Cmd {
	return func() tea.Msg {
		if url == AllFeedsURL {
			f, err := feed.FetchAll(allFeeds)
			return feedFetchedMsg{feed: f, err: err, url: AllFeedsURL}
		}
		f, err := feed.Fetch(url)
		return feedFetchedMsg{feed: f, err: err, url: url}
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}

// Update handles messages and updates the model state.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == addingFeedView {
			return m.updateAddingFeedView(msg)
		}
		if m.state == quitView {
			return m.updateQuitView(msg)
		}

		if key.Matches(msg, m.keys.Quit) {
			m.previousState = m.state
			m.state = quitView
			return m, nil
		}

		// Try to handle keys in current state helpers
		var handled bool
		switch m.state {
		case feedView:
			cmd, handled = m.handleFeedViewKeys(msg)
		case articleView:
			cmd, handled = m.handleArticleViewKeys(msg)
		case detailView:
			cmd, handled = m.handleDetailViewKeys(msg)
		}
		if handled {
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		sidebarWidth := msg.Width / 3
		mainWidth := msg.Width - sidebarWidth
		// Sidebar now has external Title (2 lines approx: Text + Padding)
		// List internal title was 2 lines?
		// We need to ensure we don't overflow.
		// Previous height was msg.Height - 3.
		// If we render Title outside, we occupy space.
		// Title + PaddingBottom(1) = 2 lines.
		// So listHeight should be msg.Height - 3 - 2 = msg.Height - 5?
		// Let's conservative decrease.
		listHeight := msg.Height - 5
		m.feedList.SetSize(sidebarWidth, listHeight)
		m.articleList.SetSize(mainWidth, listHeight)
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 5 // Leave room for header (2 lines) /footer

		// Re-size article list if needed when window resizes
		m.articleList.SetSize(mainWidth, listHeight)

	case feedFetchedMsg:
		m.handleFeedFetchedMsg(msg)
	}

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// ready boolean was unused

	switch m.state {
	case feedView:
		prevIdx := m.feedList.Index()
		m.feedList, cmd = m.feedList.Update(msg)
		if m.feedList.Index() != prevIdx {
			m.err = nil // Clear error on new selection
			if i, ok := m.feedList.SelectedItem().(*item); ok {
				m.updateArticleList(i.link)

				// Auto-fetch if no items in cache
				if len(m.articleList.Items()) == 0 {
					m.loading = true
					// Ensure spinner ticks and fetch runs
					cmds = append(cmds, tea.Batch(m.spinner.Tick, fetchFeedCmd(i.link, m.cfg.Feeds)))
				} else {
					// Stop loading if we have items (cancels previous spinner for stale requests)
					m.loading = false
				}
			}
		}
		cmds = append(cmds, cmd)
	case articleView:
		m.articleList, cmd = m.articleList.Update(msg)
		cmds = append(cmds, cmd)
	case detailView:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateAddingFeedView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "enter":
		url := m.textInput.Value()
		if url != "" {
			_ = m.cfg.AddFeed(url)
			m.refreshFeedList()
			m.textInput.Reset()
		}
		m.state = feedView
		return m, nil
	case "esc":
		m.textInput.Reset()
		m.state = feedView
		return m, nil
	}
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) updateQuitView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, tea.Quit
	case "n", "N", "esc", "q", "Q":
		m.state = m.previousState
		return m, nil
	}
	return m, nil
}

func (m *Model) handleFeedViewKeys(msg tea.KeyMsg) (tea.Cmd, bool) {
	if key.Matches(msg, m.keys.Right) || key.Matches(msg, m.keys.Open) {
		if i, ok := m.feedList.SelectedItem().(*item); ok {
			m.loading = true
			m.state = articleView
			m.articleList.ResetSelected()
			m.articleList.ResetFilter()
			return tea.Batch(m.spinner.Tick, fetchFeedCmd(i.link, m.cfg.Feeds)), true
		}
	}
	if key.Matches(msg, m.keys.AddFeed) {
		m.state = addingFeedView
		m.textInput.Reset()
		return textinput.Blink, true
	}
	if key.Matches(msg, m.keys.DeleteFeed) {
		if len(m.feedList.Items()) > 0 {
			index := m.feedList.Index()
			// Prevent deleting All Tab (index 0)
			if index == 0 {
				return nil, true
			}
			// Adjust index for All Tab
			realIndex := index - 1
			_ = m.cfg.RemoveFeed(realIndex)
			m.refreshFeedList()
		}
		return nil, true
	}
	if key.Matches(msg, m.keys.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return nil, true
	}
	return nil, false
}

func (m *Model) handleArticleViewKeys(msg tea.KeyMsg) (tea.Cmd, bool) {
	if key.Matches(msg, m.keys.Left) || key.Matches(msg, m.keys.Back) {
		m.state = feedView
		m.articleList.Title = "Articles"
		m.currentFeed = nil
		return nil, true
	}
	if key.Matches(msg, m.keys.Right) || key.Matches(msg, m.keys.Open) {
		if i, ok := m.articleList.SelectedItem().(*item); ok {
			// Mark as Read
			if _, ok := m.history[i.guid]; ok {
				m.history[i.guid].IsRead = true
				var snapshot []*history.Item
				for _, v := range m.history {
					snapshot = append(snapshot, v)
				}
				go func() { _ = m.historyMgr.Save(snapshot) }()

				// Visually update immediately
				idx := m.articleList.Index()
				i.isRead = true
				i.isRead = true
				// i.title = lipgloss.NewStyle().Faint(true).Render(i.title) // Removing direct styling
				m.articleList.SetItem(idx, i)
			}

			m.state = detailView
			text := i.content
			if text == "" {
				text = i.desc
			}
			m.viewport.SetContent(fmt.Sprintf("%s\n\n%s", i.title, text))
			m.viewport.GotoTop()
		}
		return nil, true
	}
	if key.Matches(msg, m.keys.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return nil, true
	}
	if key.Matches(msg, m.keys.Refresh) {
		if m.currentFeed != nil {
			m.loading = true
			return tea.Batch(m.spinner.Tick, fetchFeedCmd(m.currentFeed.URL, m.cfg.Feeds)), true
		}
	}
	return nil, false
}

//nolint:unparam
func (m *Model) handleDetailViewKeys(msg tea.KeyMsg) (tea.Cmd, bool) {
	if key.Matches(msg, m.keys.Left) || key.Matches(msg, m.keys.Back) {
		m.state = articleView
		return nil, true
	}
	if key.Matches(msg, m.keys.Right) || key.Matches(msg, m.keys.Open) {
		if i, ok := m.articleList.SelectedItem().(*item); ok {
			_ = openBrowser(i.link)
		}
		return nil, true
	}
	if key.Matches(msg, m.keys.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return nil, true
	}
	return nil, false
}

func (m *Model) handleFeedFetchedMsg(msg feedFetchedMsg) {
	// 1. Merge and Save History (Always, if successful)
	if msg.err == nil {
		m.loading = false // Default assumption to stop loading on success? No, handle in UI check.
		// Actually, we should only stop loading if it matches current view.
		// But existing logic merges history first.

		for _, it := range msg.feed.Items {
			guid := it.Link
			if guid == "" {
				guid = it.Title
			}

			if _, exists := m.history[guid]; !exists {
				m.history[guid] = &history.Item{
					GUID:        guid,
					Title:       it.Title,
					Description: it.Description,
					Content:     it.Content,
					Link:        it.Link,
					Published:   it.Published,
					Date:        it.Date,
					FeedTitle:   it.FeedTitle,
					FeedURL:     it.FeedURL,
					IsRead:      false,
					SavedAt:     time.Now(),
				}
			} else if m.history[guid].FeedURL == "" {
				m.history[guid].FeedURL = it.FeedURL
			}
		}

		// Save History
		var allHistory []*history.Item
		for _, v := range m.history {
			allHistory = append(allHistory, v)
		}
		go func() { _ = m.historyMgr.Save(allHistory) }()
	}

	// 2. Check if we should update the UI
	// We only update if the fetched URL matches the currently selected feed.
	currentURL := ""
	if i, ok := m.feedList.SelectedItem().(*item); ok {
		currentURL = i.link
	}

	if msg.url == currentURL {
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			// Only force state reset if we were in article view waiting?
			// But for sidebar auto-fetch we are in feedView.
			// Existing logic used m.state = feedView.
			// Just leave state as is usually safer but we can keep assignment.
			m.state = feedView
		} else {
			m.currentFeed = msg.feed
			m.articleList.Title = msg.feed.Title
			// Update the list with newly merged items
			m.updateArticleList(msg.url)
		}
	} else {
		// Stale response. Logic:
		// - History is updated (if success).
		// - UI is NOT updated.
		// - m.loading is NOT touched (keep waiting for the *correct* response if any)
	}
}

func (m *Model) updateArticleList(feedURL string) {
	// 3. Prepare Display Items using Cached History
	var displayItems []*history.Item
	for _, hItem := range m.history {
		if feedURL == AllFeedsURL {
			displayItems = append(displayItems, hItem)
		} else if hItem.FeedURL == feedURL {
			displayItems = append(displayItems, hItem)
		}
	}

	// Sort by Date Descending
	sort.Slice(displayItems, func(i, j int) bool {
		return displayItems[i].Date.After(displayItems[j].Date)
	})

	items := make([]list.Item, len(displayItems))

	for i, it := range displayItems {
		title := it.Title
		if feedURL == AllFeedsURL && it.FeedTitle != "" {
			title = fmt.Sprintf("[%s] %s", it.FeedTitle, title)
		}

		items[i] = &item{
			title:     title,
			desc:      it.Description,
			content:   it.Content,
			link:      it.Link,
			published: it.Published,
			guid:      it.GUID,
			isRead:    it.IsRead,
			feedTitle: it.FeedTitle,
			feedURL:   it.FeedURL,
		}
	}
	m.articleList.SetItems(items)

	// Set Title
	if feedURL == AllFeedsURL {
		m.articleList.Title = "All Feeds"
	} else {
		// Find feed title from config if possible, or just use URL/Placeholder
		// For now, let's look it up in cfg.Feeds if we want a nice title, but we have the feedURL.
		// The original 'item' in feedList has the title combined.
		// We can just set it to "Articles" or the Feed URL for now.
		m.articleList.Title = "Articles"
		// If we want the actual Feed Title we might need to lookup or pass it.
		// But in feedFetchedMsg we get the Feed object. Here we just have URL.
		// Simple improvement: Iterate cfg.Feeds or lookup from history map (if we had feed metadata).
		// For now simple "Articles" is safe.
	}
}

// View renders the application view.
func (m *Model) View() string {
	return view.Render(m.buildProps())
}

// OSOpenCmd allows mocking the open command
var OSOpenCmd = func(url string) *exec.Cmd {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		return nil
	}
	return exec.Command(cmd, args...) //nolint:gosec
}

func openBrowser(url string) error {
	cmd := OSOpenCmd(url)
	if cmd == nil {
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
