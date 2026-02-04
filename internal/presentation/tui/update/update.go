// Package update holds UI update logic for the TUI.
package update

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/application/usecase"
	"github.com/tesso57/reazy/internal/domain/reading"
	"github.com/tesso57/reazy/internal/presentation/tui/intent"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
)

// Deps groups external dependencies for updates.
type Deps struct {
	Subscriptions usecase.SubscriptionService
	Reading       usecase.ReadingService
	OpenBrowser   func(string) error
}

// FeedFetchedMsg is emitted after fetching feeds.
type FeedFetchedMsg struct {
	Feed *reading.Feed
	Err  error
	URL  string
}

// FetchFeedCmd creates a command to fetch feeds using the reading service.
func FetchFeedCmd(readingSvc usecase.ReadingService, url string, feeds []string) tea.Cmd {
	allFeeds := append([]string(nil), feeds...)
	trimmed := strings.TrimSpace(url)
	return func() tea.Msg {
		f, err := readingSvc.FetchFeed(trimmed, allFeeds)
		return FeedFetchedMsg{Feed: f, Err: err, URL: trimmed}
	}
}

// HandleKeyMsg processes key input based on the current session.
func HandleKeyMsg(s *state.ModelState, msg tea.KeyMsg, deps Deps) (tea.Cmd, bool) {
	if s.Session == state.AddingFeedView {
		return handleAddingFeedView(s, msg, deps)
	}
	if s.Session == state.QuitView {
		return handleQuitView(s, msg)
	}
	if s.Session == state.DeleteFeedView {
		return handleDeleteFeedView(s, msg, deps)
	}

	parsed := intent.FromKeyMsg(msg, s.Keys)
	if parsed.Type == intent.Quit {
		s.Previous = s.Session
		s.Session = state.QuitView
		return nil, true
	}

	switch s.Session {
	case state.FeedView:
		return handleFeedViewIntent(s, parsed, deps)
	case state.ArticleView:
		return handleArticleViewIntent(s, parsed, deps)
	case state.DetailView:
		return handleDetailViewIntent(s, parsed, deps)
	default:
		return nil, false
	}
}

// HandleWindowSize updates layout sizing based on terminal size.
func HandleWindowSize(s *state.ModelState, msg tea.WindowSizeMsg) {
	s.Width = msg.Width
	s.Height = msg.Height

	UpdateListSizes(s)
}

// HandleFeedFetchedMsg merges history and updates lists if applicable.
func HandleFeedFetchedMsg(s *state.ModelState, msg FeedFetchedMsg, deps Deps) {
	if msg.Err == nil {
		s.Loading = false
		deps.Reading.MergeHistory(s.History, msg.Feed)
		go func() { _ = deps.Reading.SaveHistory(s.History) }()
	}

	currentURL := ""
	if i, ok := s.FeedList.SelectedItem().(*presenter.Item); ok {
		currentURL = i.Link
	}

	if msg.URL == currentURL {
		s.Loading = false
		if msg.Err != nil {
			s.Err = msg.Err
			s.Session = state.FeedView
			return
		}
		s.CurrentFeed = msg.Feed
		presenter.ApplyArticleList(&s.ArticleList, s.History, msg.URL)
		UpdateListSizes(s)
	}
}

func handleAddingFeedView(s *state.ModelState, msg tea.KeyMsg, deps Deps) (tea.Cmd, bool) {
	switch msg.String() {
	case "enter":
		url := s.TextInput.Value()
		if url != "" {
			feeds, err := deps.Subscriptions.Add(url)
			if err != nil {
				s.Err = err
			} else {
				s.Feeds = feeds
				presenter.ApplyFeedList(&s.FeedList, s.Feeds)
				UpdateListSizes(s)
			}
			s.TextInput.Reset()
		}
		s.Session = state.FeedView
		return nil, true
	case "esc":
		s.TextInput.Reset()
		s.Session = state.FeedView
		return nil, true
	}

	var cmd tea.Cmd
	s.TextInput, cmd = s.TextInput.Update(msg)
	return cmd, true
}

func handleQuitView(s *state.ModelState, msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "y", "Y":
		return tea.Quit, true
	case "n", "N", "esc", "q", "Q":
		s.Session = s.Previous
		return nil, true
	}
	return nil, true
}

func handleDeleteFeedView(s *state.ModelState, msg tea.KeyMsg, deps Deps) (tea.Cmd, bool) {
	switch msg.String() {
	case "y", "Y":
		index := s.FeedList.Index()
		if index > 0 {
			realIndex := index - 1
			feeds, err := deps.Subscriptions.Remove(realIndex)
			if err != nil {
				s.Err = err
			} else {
				s.Feeds = feeds
				presenter.ApplyFeedList(&s.FeedList, s.Feeds)
				UpdateListSizes(s)
			}
		}
		s.Session = state.FeedView
		return nil, true
	case "n", "N", "esc", "q", "Q":
		s.Session = state.FeedView
		return nil, true
	}
	return nil, true
}

func handleFeedViewIntent(s *state.ModelState, in intent.Intent, deps Deps) (tea.Cmd, bool) {
	switch in.Type {
	case intent.Open:
		if i, ok := s.FeedList.SelectedItem().(*presenter.Item); ok {
			s.Loading = true
			s.Session = state.ArticleView
			s.ArticleList.ResetSelected()
			s.ArticleList.ResetFilter()
			return tea.Batch(s.Spinner.Tick, FetchFeedCmd(deps.Reading, i.Link, s.Feeds)), true
		}
	case intent.AddFeed:
		s.Session = state.AddingFeedView
		s.TextInput.Reset()
		return textinput.Blink, true
	case intent.DeleteFeed:
		if len(s.FeedList.Items()) > 0 {
			index := s.FeedList.Index()
			if index == 0 {
				return nil, true
			}
			s.Session = state.DeleteFeedView
		}
		return nil, true
	case intent.ToggleHelp:
		s.Help.ShowAll = !s.Help.ShowAll
		return nil, true
	}
	return nil, false
}

func handleArticleViewIntent(s *state.ModelState, in intent.Intent, deps Deps) (tea.Cmd, bool) {
	switch in.Type {
	case intent.Back:
		s.Session = state.FeedView
		s.ArticleList.Title = "Articles"
		s.CurrentFeed = nil
		return nil, true
	case intent.Open:
		if i, ok := s.ArticleList.SelectedItem().(*presenter.Item); ok {
			if s.History.MarkRead(i.GUID) {
				go func() { _ = deps.Reading.SaveHistory(s.History) }()

				idx := s.ArticleList.Index()
				i.Read = true
				s.ArticleList.SetItem(idx, i)
			}

			s.Session = state.DetailView
			text := i.Content
			if text == "" {
				text = i.Desc
			}
			s.Viewport.SetContent(fmt.Sprintf("%s\n\n%s", i.TitleText, text))
			s.Viewport.GotoTop()
		}
		return nil, true
	case intent.ToggleHelp:
		s.Help.ShowAll = !s.Help.ShowAll
		return nil, true
	case intent.Refresh:
		if s.CurrentFeed != nil {
			s.Loading = true
			return tea.Batch(s.Spinner.Tick, FetchFeedCmd(deps.Reading, s.CurrentFeed.URL, s.Feeds)), true
		}
	case intent.Bookmark:
		if i, ok := s.ArticleList.SelectedItem().(*presenter.Item); ok {
			_ = deps.Reading.ToggleBookmark(s.History, i.GUID)

			// Update the item in the list immediately
			idx := s.ArticleList.Index()
			i.Bookmarked = !i.Bookmarked
			s.ArticleList.SetItem(idx, i)
			return nil, true
		}
	}
	return nil, false
}

func handleDetailViewIntent(s *state.ModelState, in intent.Intent, deps Deps) (tea.Cmd, bool) {
	switch in.Type {
	case intent.Back:
		s.Session = state.ArticleView
		return nil, true
	case intent.Open:
		if i, ok := s.ArticleList.SelectedItem().(*presenter.Item); ok {
			_ = deps.OpenBrowser(i.Link)
		}
		return nil, true
	case intent.ToggleHelp:
		s.Help.ShowAll = !s.Help.ShowAll
		return nil, true
	}
	return nil, false
}
