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
	"github.com/tesso57/reazy/internal/domain/subscription"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
	"github.com/tesso57/reazy/internal/presentation/tui/update"
	"github.com/tesso57/reazy/internal/presentation/tui/view"
	listview "github.com/tesso57/reazy/internal/presentation/tui/view/list"
)

// Model represents the main application state.
type Model struct {
	settings      settings.Settings
	subscriptions *usecase.SubscriptionService
	reading       *usecase.ReadingService
	insights      *usecase.InsightService
	newsDigests   *usecase.NewsDigestService
	feedGrouping  *usecase.FeedGroupingService
	state         *state.ModelState
}

// NewModel creates a new application model.
func NewModel(cfg settings.Settings, subscriptions *usecase.SubscriptionService, readingSvc *usecase.ReadingService) *Model {
	return NewModelWithServices(
		cfg,
		subscriptions,
		readingSvc,
		usecase.NewInsightService(nil, nil),
		usecase.NewNewsDigestService(nil, nil, nil),
		usecase.NewFeedGroupingService(nil),
	)
}

// NewModelWithInsights creates a new application model with AI insights support.
func NewModelWithInsights(cfg settings.Settings, subscriptions *usecase.SubscriptionService, readingSvc *usecase.ReadingService, insightSvc *usecase.InsightService) *Model {
	return NewModelWithServices(cfg, subscriptions, readingSvc, insightSvc, usecase.NewNewsDigestService(nil, nil, nil), usecase.NewFeedGroupingService(nil))
}

// NewModelWithServices creates a new application model with all optional AI services.
func NewModelWithServices(
	cfg settings.Settings,
	subscriptions *usecase.SubscriptionService,
	readingSvc *usecase.ReadingService,
	insightSvc *usecase.InsightService,
	newsDigestSvc *usecase.NewsDigestService,
	feedGroupingSvc *usecase.FeedGroupingService,
) *Model {
	return new(Model{
		settings:      cfg,
		subscriptions: subscriptions,
		reading:       readingSvc,
		insights:      insightSvc,
		newsDigests:   newsDigestSvc,
		feedGrouping:  feedGroupingSvc,
		state:         newModelState(cfg, readingSvc),
	})
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
		cmds = append(cmds, update.HandleFeedFetchedMsg(m.state, msg, m.deps()))
	case update.NewsDigestGeneratedMsg:
		update.HandleNewsDigestGeneratedMsg(m.state, msg, m.deps())
	case update.FeedGroupingCompletedMsg:
		update.HandleFeedGroupingCompletedMsg(m.state, msg)
	case update.InsightGeneratedMsg:
		update.HandleInsightGeneratedMsg(m.state, msg, m.deps())
	case update.ArticleDetailLoadedMsg:
		cmds = append(cmds, update.HandleArticleDetailLoadedMsg(m.state, msg, m.deps()))
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
				if i.IsSectionHeader() {
					m.state.Loading = false
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}

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
	case state.NewsTopicView:
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
		NewsDigests:   m.newsDigests,
		FeedGrouping:  m.feedGrouping,
		OpenBrowser:   openBrowser,
	}
}

func newModelState(cfg settings.Settings, readingSvc *usecase.ReadingService) *state.ModelState {
	st := new(state.ModelState{
		Session:             state.FeedView,
		FeedList:            newFeedList(cfg),
		ArticleList:         newArticleList(),
		TextInput:           newTextInput(),
		Viewport:            newViewport(),
		Help:                help.New(),
		Spinner:             newSpinner(),
		Keys:                state.NewKeyMap(cfg.KeyMap),
		History:             loadHistory(readingSvc),
		Feeds:               append([]string(nil), cfg.FlattenedFeeds()...),
		FeedGroups:          cloneFeedGroups(cfg.FeedGroups),
		ShowAISummary:       true,
		DetailParentSession: state.ArticleView,
	})

	st.FeedList.KeyMap.PrevPage = st.Keys.UpPage
	st.FeedList.KeyMap.NextPage = st.Keys.DownPage
	st.ArticleList.KeyMap.PrevPage = st.Keys.UpPage
	st.ArticleList.KeyMap.NextPage = st.Keys.DownPage

	presenter.ApplyFeedList(&st.FeedList, st.Feeds, st.FeedGroups)
	presenter.ApplyArticleList(&st.ArticleList, st.History, reading.AllFeedsURL)

	return st
}

func cloneFeedGroups(groups []subscription.FeedGroup) []subscription.FeedGroup {
	if len(groups) == 0 {
		return nil
	}
	out := make([]subscription.FeedGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, subscription.FeedGroup{
			Name:  group.Name,
			Feeds: append([]string(nil), group.Feeds...),
		})
	}
	return out
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

func loadHistory(readingSvc *usecase.ReadingService) *reading.History {
	hist, _ := readingSvc.LoadHistoryMetadata()
	if hist == nil {
		hist = reading.NewHistory(nil)
	}
	return hist
}
