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
)

const AllFeedsURL = "internal://all"

type Model struct {
	cfg         *config.Config
	state       sessionState
	feedList    list.Model
	articleList list.Model
	textInput   textinput.Model
	viewport    viewport.Model
	help        help.Model
	spinner     spinner.Model
	loading     bool
	keys        KeyMap
	width       int
	height      int
	currentFeed *feed.Feed
	err         error

	historyMgr *history.Manager
	history    map[string]*history.HistoryItem // GUID -> Item
}

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

func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Back, k.Open}
}

func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Top, k.Bottom, k.UpPage, k.DownPage},
		{k.Open, k.Back, k.Quit},
		{k.AddFeed, k.DeleteFeed, k.Refresh, k.Help},
	}
}

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

func NewModel(cfg *config.Config) *Model {
	l := list.New([]list.Item{}, delegate.NewFeedDelegate(lipgloss.Color(cfg.Theme.FeedName)), 0, 0)
	l.Title = "Reazy Feeds"
	l.SetShowHelp(false)
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
		hist = make(map[string]*history.HistoryItem)
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
}

func fetchFeedCmd(url string, allFeeds []string) tea.Cmd {
	return func() tea.Msg {
		if url == AllFeedsURL {
			f, err := feed.FetchAll(allFeeds)
			return feedFetchedMsg{feed: f, err: err}
		}
		f, err := feed.Fetch(url)
		return feedFetchedMsg{feed: f, err: err}
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == addingFeedView {
			return m.updateAddingFeedView(msg)
		}

		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
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
		listHeight := msg.Height - 3
		m.feedList.SetSize(sidebarWidth, listHeight)
		m.articleList.SetSize(mainWidth, listHeight)
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 5 // Leave room for header (2 lines) /footer

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
		m.feedList, cmd = m.feedList.Update(msg)
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
				var snapshot []*history.HistoryItem
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
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		m.state = feedView
	} else {
		m.currentFeed = msg.feed
		m.articleList.Title = msg.feed.Title

		// 1. Merge fetched items into history
		for _, it := range msg.feed.Items {
			guid := it.Link
			if guid == "" {
				guid = it.Title
			}

			if _, exists := m.history[guid]; !exists {
				m.history[guid] = &history.HistoryItem{
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
			} else {
				// Update FeedURL if missing (migration)
				if m.history[guid].FeedURL == "" {
					m.history[guid].FeedURL = it.FeedURL
				}
			}
		}

		// 2. Save History
		// Convert map to slice for saving
		var allHistory []*history.HistoryItem
		for _, v := range m.history {
			allHistory = append(allHistory, v)
		}
		// Best effort save
		go func() { _ = m.historyMgr.Save(allHistory) }()

		// 3. Prepare Display Items
		var displayItems []*history.HistoryItem
		for _, hItem := range m.history {
			if m.currentFeed.URL == AllFeedsURL {
				displayItems = append(displayItems, hItem)
			} else {
				if hItem.FeedURL == m.currentFeed.URL {
					displayItems = append(displayItems, hItem)
				}
			}
		}

		// Sort by Date Descending
		sort.Slice(displayItems, func(i, j int) bool {
			return displayItems[i].Date.After(displayItems[j].Date)
		})

		items := make([]list.Item, len(displayItems))

		for i, it := range displayItems {
			title := it.Title
			if m.currentFeed.URL == AllFeedsURL && it.FeedTitle != "" {
				// We format the title here, but without coloring.
				// Coloring should happen in Delegate if possible, but for mixed content string
				// it's easier to prepare the string here if we want to avoid complex delegate logic.
				// However, refactoring goal is to remove lipgloss.
				// So we produce a clean string here: "[FeedName] Title"
				title = fmt.Sprintf("[%s] %s", it.FeedTitle, title)
			}

			// Faint styling for read items is handled by Delegate via IsRead check.

			items[i] = &item{
				title:     title, // Plain text title
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
	}
}

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
	return exec.Command(cmd, args...)
}

func openBrowser(url string) error {
	cmd := OSOpenCmd(url)
	if cmd == nil {
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
