package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/application/usecase"
	"github.com/tesso57/reazy/internal/domain/reading"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
	"github.com/tesso57/reazy/internal/presentation/tui/update"
	"github.com/tesso57/reazy/internal/presentation/tui/view"
	listview "github.com/tesso57/reazy/internal/presentation/tui/view/list"
)

// Model represents the main application state.
type Model struct {
	settings      settings.Settings
	subscriptions usecase.SubscriptionService
	reading       usecase.ReadingService
	insights      usecase.InsightService
	state         *state.ModelState
}

// NewModel creates a new application model.
func NewModel(cfg settings.Settings, subscriptions usecase.SubscriptionService, readingSvc usecase.ReadingService) *Model {
	return NewModelWithInsights(cfg, subscriptions, readingSvc, usecase.NewInsightService(nil, nil))
}

// NewModelWithInsights creates a new application model with AI insights support.
func NewModelWithInsights(cfg settings.Settings, subscriptions usecase.SubscriptionService, readingSvc usecase.ReadingService, insightSvc usecase.InsightService) *Model {
	return &Model{
		settings:      cfg,
		subscriptions: subscriptions,
		reading:       readingSvc,
		insights:      insightSvc,
		state:         newModelState(cfg, readingSvc),
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.state.Spinner.Tick, textinput.Blink)
}

// Update handles messages and updates the model state.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd, handled := update.HandleKeyMsg(m.state, msg, m.deps())
		if handled {
			update.UpdateListSizes(m.state)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		update.HandleWindowSize(m.state, msg)
	case update.FeedFetchedMsg:
		update.HandleFeedFetchedMsg(m.state, msg, m.deps())
	case update.InsightGeneratedMsg:
		update.HandleInsightGeneratedMsg(m.state, msg, m.deps())
	}

	if m.state.Loading {
		m.state.Spinner, cmd = m.state.Spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state.Session {
	case state.FeedView:
		prevIdx := m.state.FeedList.Index()
		m.state.FeedList, cmd = m.state.FeedList.Update(msg)
		if m.state.FeedList.Index() != prevIdx {
			m.state.Err = nil
			if i, ok := m.state.FeedList.SelectedItem().(*presenter.Item); ok {
				presenter.ApplyArticleList(&m.state.ArticleList, m.state.History, i.Link)
				update.UpdateListSizes(m.state)

				if len(m.state.ArticleList.Items()) == 0 {
					m.state.Loading = true
					cmds = append(cmds, tea.Batch(m.state.Spinner.Tick, update.FetchFeedCmd(m.reading, i.Link, m.state.Feeds)))
				} else {
					m.state.Loading = false
				}
			}
		}
		cmds = append(cmds, cmd)
	case state.ArticleView:
		m.state.ArticleList, cmd = m.state.ArticleList.Update(msg)
		cmds = append(cmds, cmd)
	case state.DetailView:
		m.state.Viewport, cmd = m.state.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the application view.
func (m *Model) View() string {
	return view.Render(m.buildProps())
}

func (m *Model) deps() update.Deps {
	return update.Deps{
		Subscriptions: m.subscriptions,
		Reading:       m.reading,
		Insights:      m.insights,
		OpenBrowser:   openBrowser,
	}
}

func newModelState(cfg settings.Settings, readingSvc usecase.ReadingService) *state.ModelState {
	st := &state.ModelState{
		Session:       state.FeedView,
		FeedList:      newFeedList(cfg),
		ArticleList:   newArticleList(),
		TextInput:     newTextInput(),
		Viewport:      newViewport(),
		Help:          help.New(),
		Spinner:       newSpinner(),
		Keys:          state.NewKeyMap(cfg.KeyMap),
		History:       loadHistory(readingSvc),
		Feeds:         append([]string(nil), cfg.Feeds...),
		ShowAISummary: true,
	}

	st.FeedList.KeyMap.PrevPage = st.Keys.UpPage
	st.FeedList.KeyMap.NextPage = st.Keys.DownPage
	st.ArticleList.KeyMap.PrevPage = st.Keys.UpPage
	st.ArticleList.KeyMap.NextPage = st.Keys.DownPage

	presenter.ApplyFeedList(&st.FeedList, st.Feeds)
	presenter.ApplyArticleList(&st.ArticleList, st.History, reading.AllFeedsURL)

	return st
}

func newFeedList(cfg settings.Settings) list.Model {
	l := list.New([]list.Item{}, listview.NewFeedDelegate(lipgloss.Color(cfg.Theme.FeedName)), 0, 0)
	l.Title = "Reazy Feeds"
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()
	return l
}

func newArticleList() list.Model {
	l := list.New([]list.Item{}, listview.NewArticleDelegate(), 0, 0)
	l.Title = "Articles"
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	return l
}

func newTextInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "https://example.com/feed.xml (RSS/Atom)"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40
	return ti
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return s
}

func newViewport() viewport.Model {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1)
	return vp
}

func loadHistory(readingSvc usecase.ReadingService) *reading.History {
	hist, _ := readingSvc.LoadHistory()
	if hist == nil {
		hist = reading.NewHistory(nil)
	}
	return hist
}
