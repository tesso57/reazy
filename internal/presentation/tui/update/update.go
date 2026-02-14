// Package update holds UI update logic for the TUI.
package update

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
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
	Subscriptions *usecase.SubscriptionService
	Reading       *usecase.ReadingService
	Insights      *usecase.InsightService
	NewsDigests   *usecase.NewsDigestService
	OpenBrowser   func(string) error
}

// FeedFetchedMsg is emitted after fetching feeds.
type FeedFetchedMsg struct {
	Feed   *reading.Feed
	Report usecase.FeedFetchReport
	Err    error
	URL    string
}

// InsightGeneratedMsg is emitted after generating AI insight for an article.
type InsightGeneratedMsg struct {
	GUID    string
	Insight usecase.Insight
	Err     error
}

// ArticleDetailLoadedMsg is emitted after loading one hydrated history item.
type ArticleDetailLoadedMsg struct {
	GUID   string
	Item   *reading.HistoryItem
	Err    error
	Silent bool
}

// NewsDigestGeneratedMsg is emitted after generating daily news digest topics.
type NewsDigestGeneratedMsg struct {
	DateKey   string
	Items     []*reading.HistoryItem
	UsedCache bool
	Force     bool
	Err       error
}

// FetchFeedCmd creates a command to fetch feeds using the reading service.
func FetchFeedCmd(readingSvc *usecase.ReadingService, url string, feeds []string) tea.Cmd {
	allFeeds := append([]string(nil), feeds...)
	trimmed := strings.TrimSpace(url)
	return func() tea.Msg {
		f, report, err := readingSvc.FetchFeed(trimmed, allFeeds)
		return FeedFetchedMsg{Feed: f, Report: report, Err: err, URL: trimmed}
	}
}

// GenerateInsightCmd creates a command to generate AI summary/tags for one article.
func GenerateInsightCmd(insightSvc *usecase.InsightService, guid string, req usecase.InsightRequest) tea.Cmd {
	return func() tea.Msg {
		insight, err := insightSvc.Generate(context.Background(), req)
		return InsightGeneratedMsg{
			GUID:    guid,
			Insight: insight,
			Err:     err,
		}
	}
}

// GenerateDailyNewsDigestCmd creates a command to build today's digest topics.
func GenerateDailyNewsDigestCmd(newsSvc *usecase.NewsDigestService, readingSvc *usecase.ReadingService, history *reading.History, feeds []string, force bool) tea.Cmd {
	historySnapshot := cloneHistoryForDigest(history)
	feedSnapshot := append([]string(nil), feeds...)
	return func() tea.Msg {
		if newsSvc == nil {
			return NewsDigestGeneratedMsg{
				Force: force,
				Err:   fmt.Errorf("codex integration is disabled"),
			}
		}
		if readingSvc == nil {
			return NewsDigestGeneratedMsg{
				Force: force,
				Err:   fmt.Errorf("reading service is not configured"),
			}
		}
		dateKey := newsSvc.TodayDateKey()
		todayArticles, err := readingSvc.LoadTodayArticles(dateKey, feedSnapshot, 60, time.Local)
		if err != nil {
			return NewsDigestGeneratedMsg{
				DateKey: dateKey,
				Force:   force,
				Err:     err,
			}
		}
		for _, article := range todayArticles {
			historySnapshot.UpsertItem(article)
		}
		digest, err := newsSvc.BuildDaily(context.Background(), historySnapshot, feedSnapshot, force)
		return NewsDigestGeneratedMsg{
			DateKey:   digest.DateKey,
			Items:     digest.Items,
			UsedCache: digest.UsedCache,
			Force:     force,
			Err:       err,
		}
	}
}

// LoadArticleDetailCmd loads one article body from persistence.
func LoadArticleDetailCmd(readingSvc *usecase.ReadingService, guid string, silent bool) tea.Cmd {
	guid = strings.TrimSpace(guid)
	return func() tea.Msg {
		item, err := readingSvc.LoadHistoryItem(guid)
		return ArticleDetailLoadedMsg{
			GUID:   guid,
			Item:   item,
			Err:    err,
			Silent: silent,
		}
	}
}

func cloneHistoryForDigest(history *reading.History) *reading.History {
	if history == nil {
		return reading.NewHistory(nil)
	}
	cloned := make(map[string]*reading.HistoryItem, len(history.Items()))
	for guid, item := range history.Items() {
		if item == nil {
			continue
		}
		copyItem := *item
		copyItem.AITags = append([]string(nil), item.AITags...)
		copyItem.RelatedGUIDs = append([]string(nil), item.RelatedGUIDs...)
		cloned[guid] = &copyItem
	}
	return reading.NewHistory(cloned)
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
	if handleFilterExitWithJJ(s, msg) {
		return nil, true
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
	case state.NewsTopicView:
		return handleNewsTopicViewIntent(s, parsed, deps)
	case state.DetailView:
		return handleDetailViewIntent(s, parsed, deps)
	default:
		return nil, false
	}
}

func handleFilterExitWithJJ(s *state.ModelState, msg tea.KeyMsg) bool {
	activeList, ok := activeListForFiltering(s)
	if !ok || activeList.FilterState() != list.Filtering {
		s.PendingJJExit = false
		return false
	}
	if msg.String() != "j" {
		s.PendingJJExit = false
		return false
	}
	if s.PendingJJExit {
		activeList.ResetFilter()
		s.PendingJJExit = false
		return true
	}
	s.PendingJJExit = true
	return false
}

func activeListForFiltering(s *state.ModelState) (*list.Model, bool) {
	switch s.Session {
	case state.FeedView:
		return &s.FeedList, true
	case state.ArticleView:
		return &s.ArticleList, true
	case state.NewsTopicView:
		return &s.ArticleList, true
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
func HandleFeedFetchedMsg(s *state.ModelState, msg FeedFetchedMsg, deps Deps) tea.Cmd {
	if msg.Err == nil {
		s.Loading = false
		if err := deps.Reading.MergeHistory(s.History, msg.Feed); err != nil {
			s.Err = err
		}
		s.StatusMessage = feedFetchStatusMessage(msg.Report)
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
			return nil
		}
		s.CurrentFeed = msg.Feed
		presenter.ApplyArticleList(&s.ArticleList, s.History, msg.URL)
		UpdateListSizes(s)
		if msg.URL == reading.NewsURL {
			force := s.ForceNewsDigestRefresh
			s.ForceNewsDigestRefresh = false
			s.Loading = true
			s.Err = nil
			s.AIStatus = "AI: generating daily news..."
			return tea.Batch(
				s.Spinner.Tick,
				GenerateDailyNewsDigestCmd(deps.NewsDigests, deps.Reading, s.History, s.Feeds, force),
			)
		}
	}
	return nil
}

// HandleNewsDigestGeneratedMsg applies generated digest items to history and current news list.
func HandleNewsDigestGeneratedMsg(s *state.ModelState, msg NewsDigestGeneratedMsg, deps Deps) {
	s.Loading = false
	defer UpdateListSizes(s)

	if msg.Err != nil {
		s.AIStatus = fmt.Sprintf("AI: daily news failed (%s)", strings.TrimSpace(msg.Err.Error()))
		s.Err = msg.Err
		if s.CurrentFeed != nil && s.CurrentFeed.URL == reading.NewsURL {
			presenter.ApplyArticleList(&s.ArticleList, s.History, reading.NewsURL)
		}
		return
	}

	s.Err = nil
	if !msg.UsedCache {
		if err := deps.Reading.ReplaceDigestItemsByDate(s.History, msg.DateKey, msg.Items); err != nil {
			s.Err = err
		}
		s.AIStatus = fmt.Sprintf("AI: daily news updated %s", time.Now().Format("2006-01-02 15:04"))
	} else {
		s.AIStatus = fmt.Sprintf("AI: using daily news cache (%s)", msg.DateKey)
	}

	if s.CurrentFeed != nil && s.CurrentFeed.URL == reading.NewsURL {
		presenter.ApplyArticleList(&s.ArticleList, s.History, reading.NewsURL)
	}
}

// HandleInsightGeneratedMsg applies AI-generated summary/tags to history and visible items.
func HandleInsightGeneratedMsg(s *state.ModelState, msg InsightGeneratedMsg, deps Deps) {
	s.Loading = false
	defer UpdateListSizes(s)
	if msg.Err != nil {
		s.AIStatus = fmt.Sprintf("AI: generation failed (%s)", strings.TrimSpace(msg.Err.Error()))
		return
	}

	updatedAt, ok, err := deps.Reading.ApplyInsight(s.History, msg.GUID, msg.Insight)
	if err != nil {
		s.AIStatus = fmt.Sprintf("AI: save failed (%s)", strings.TrimSpace(err.Error()))
		return
	}
	if !ok {
		s.AIStatus = "AI: failed to attach generated insight"
		return
	}
	s.AIStatus = fmt.Sprintf("AI: updated %s", updatedAt.Format("2006-01-02 15:04"))

	for idx, listItem := range s.ArticleList.Items() {
		item, ok := listItem.(*presenter.Item)
		if !ok || item.GUID != msg.GUID {
			continue
		}
		item.AISummary = msg.Insight.Summary
		item.AITags = append([]string(nil), msg.Insight.Tags...)
		item.AIUpdatedAt = updatedAt
		s.ArticleList.SetItem(idx, item)
		if s.Session == state.DetailView && s.ArticleList.Index() == idx {
			refreshDetailViewport(s, item)
		}
		break
	}
}

// HandleArticleDetailLoadedMsg applies hydrated article payload to state.
func HandleArticleDetailLoadedMsg(s *state.ModelState, msg ArticleDetailLoadedMsg, deps Deps) tea.Cmd {
	if msg.Err != nil {
		s.Loading = false
		if !msg.Silent {
			s.Err = msg.Err
		}
		s.PendingInsightGUID = ""
		return nil
	}
	if msg.Item != nil {
		msg.Item.BodyHydrated = true
		s.History.UpsertItem(msg.Item)
		applyHydratedItemToList(&s.ArticleList, msg.Item)

		if s.Session == state.DetailView {
			if selected, ok := selectedActionableArticleItem(s); ok && selected.GUID == msg.Item.GUID {
				refreshDetailViewport(s, selected)
			}
		}
	}

	if s.PendingInsightGUID != "" && s.PendingInsightGUID == msg.GUID {
		s.PendingInsightGUID = ""
		s.Loading = true
		s.AIStatus = "AI: generating summary and tags..."
		if selected, ok := selectedActionableArticleItem(s); ok && selected.GUID == msg.GUID {
			return tea.Batch(
				s.Spinner.Tick,
				GenerateInsightCmd(deps.Insights, selected.GUID, buildInsightRequest(selected)),
			)
		}
	}

	s.Loading = false
	return nil
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
		if item, ok := selectedFeedItem(s); ok && !reading.IsVirtualFeedURL(item.Link) {
			if realIndex, ok := subscriptionIndexByURL(s.Feeds, item.Link); ok {
				feeds, err := deps.Subscriptions.Remove(realIndex)
				if err != nil {
					s.Err = err
				} else {
					s.Feeds = feeds
					presenter.ApplyFeedList(&s.FeedList, s.Feeds)
					UpdateListSizes(s)
				}
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
		if item, ok := selectedFeedItem(s); ok {
			if reading.IsVirtualFeedURL(item.Link) {
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
		if i, ok := selectedActionableArticleItem(s); ok {
			if i.IsNewsDigest() {
				enterNewsTopicView(s, i)
				return nil, true
			}
			if err := deps.Reading.MarkRead(s.History, i.GUID); err == nil {
				idx := s.ArticleList.Index()
				i.Read = true
				s.ArticleList.SetItem(idx, i)
			}

			s.DetailParentSession = state.ArticleView
			s.Session = state.DetailView
			if !i.BodyHydrated {
				i.Content = ""
				refreshDetailViewport(s, i)
				s.Loading = true
				return tea.Batch(
					s.Spinner.Tick,
					LoadArticleDetailCmd(deps.Reading, i.GUID, true),
				), true
			}
			refreshDetailViewport(s, i)
		}
		return nil, true
	case intent.ToggleHelp:
		s.Help.ShowAll = !s.Help.ShowAll
		return nil, true
	case intent.Refresh:
		if s.CurrentFeed != nil {
			if s.CurrentFeed.URL == reading.NewsURL {
				s.ForceNewsDigestRefresh = true
			}
			s.Loading = true
			return tea.Batch(s.Spinner.Tick, FetchFeedCmd(deps.Reading, s.CurrentFeed.URL, s.Feeds)), true
		}
	case intent.Bookmark:
		if i, ok := selectedActionableArticleItem(s); ok {
			_ = deps.Reading.ToggleBookmark(s.History, i.GUID)

			// Update the item in the list immediately
			idx := s.ArticleList.Index()
			i.Bookmarked = !i.Bookmarked
			s.ArticleList.SetItem(idx, i)
			return nil, true
		}
	case intent.Summarize:
		return startInsightGenerationForSelection(s, deps), true
	case intent.ToggleSummary:
		return nil, true
	}
	return nil, false
}

func handleNewsTopicViewIntent(s *state.ModelState, in intent.Intent, deps Deps) (tea.Cmd, bool) {
	switch in.Type {
	case intent.Back:
		s.Session = state.ArticleView
		presenter.ApplyArticleList(&s.ArticleList, s.History, reading.NewsURL)
		selectArticleItemByGUID(&s.ArticleList, s.NewsTopicDigestGUID)
		return nil, true
	case intent.Open:
		if i, ok := selectedActionableArticleItem(s); ok {
			if err := deps.Reading.MarkRead(s.History, i.GUID); err == nil {
				idx := s.ArticleList.Index()
				i.Read = true
				s.ArticleList.SetItem(idx, i)
			}
			s.DetailParentSession = state.NewsTopicView
			s.Session = state.DetailView
			if !i.BodyHydrated {
				i.Content = ""
				refreshDetailViewport(s, i)
				s.Loading = true
				return tea.Batch(
					s.Spinner.Tick,
					LoadArticleDetailCmd(deps.Reading, i.GUID, true),
				), true
			}
			refreshDetailViewport(s, i)
		}
		return nil, true
	case intent.Refresh:
		if s.CurrentFeed != nil && s.CurrentFeed.URL == reading.NewsURL {
			s.ForceNewsDigestRefresh = true
			s.Session = state.ArticleView
			s.Loading = true
			return tea.Batch(s.Spinner.Tick, FetchFeedCmd(deps.Reading, s.CurrentFeed.URL, s.Feeds)), true
		}
		return nil, true
	case intent.ToggleHelp:
		s.Help.ShowAll = !s.Help.ShowAll
		return nil, true
	}
	return nil, false
}

func handleDetailViewIntent(s *state.ModelState, in intent.Intent, deps Deps) (tea.Cmd, bool) {
	switch in.Type {
	case intent.Back:
		if s.DetailParentSession == state.NewsTopicView || s.DetailParentSession == state.ArticleView {
			s.Session = s.DetailParentSession
		} else {
			s.Session = state.ArticleView
		}
		return nil, true
	case intent.Open:
		if i, ok := selectedActionableArticleItem(s); ok {
			_ = deps.OpenBrowser(i.Link)
		}
		return nil, true
	case intent.ToggleHelp:
		s.Help.ShowAll = !s.Help.ShowAll
		return nil, true
	case intent.Summarize:
		return startInsightGenerationForSelection(s, deps), true
	case intent.ToggleSummary:
		s.ShowAISummary = !s.ShowAISummary
		if i, ok := selectedActionableArticleItem(s); ok {
			refreshDetailViewport(s, i)
		}
		return nil, true
	}
	return nil, false
}

func buildInsightRequest(item *presenter.Item) usecase.InsightRequest {
	if item == nil {
		return usecase.InsightRequest{}
	}
	title := item.RawTitle
	if title == "" {
		title = item.TitleText
	}
	return usecase.InsightRequest{
		Title:       title,
		Description: item.Desc,
		Content:     item.Content,
		Link:        item.Link,
		Published:   item.Published,
		FeedTitle:   item.FeedTitleText,
	}
}

func refreshDetailViewport(s *state.ModelState, item *presenter.Item) {
	if s == nil {
		return
	}
	wrapWidth := detailWrapWidth(s)
	s.Viewport.SetContent(buildDetailContentForWidth(item, s.ShowAISummary, wrapWidth))
	s.Viewport.GotoTop()
}

func detailWrapWidth(s *state.ModelState) int {
	if s == nil {
		return 0
	}
	viewportContentWidth := s.Viewport.Width - s.Viewport.Style.GetHorizontalFrameSize()
	if viewportContentWidth > 0 {
		return viewportContentWidth
	}
	// Fallback for early calls before first resize.
	mainContentWidth := s.ArticleList.Width() - 1 // main view has left padding of 1
	return clampMin(mainContentWidth-s.Viewport.Style.GetHorizontalFrameSize(), 1)
}

func startInsightGenerationForSelection(s *state.ModelState, deps Deps) tea.Cmd {
	item, ok := selectedActionableArticleItem(s)
	if !ok {
		return nil
	}
	if item.IsNewsDigest() {
		return nil
	}

	if !item.BodyHydrated {
		s.Loading = true
		s.Err = nil
		s.PendingInsightGUID = item.GUID
		s.AIStatus = "AI: loading article content..."
		return tea.Batch(
			s.Spinner.Tick,
			LoadArticleDetailCmd(deps.Reading, item.GUID, false),
		)
	}

	s.Loading = true
	s.Err = nil
	s.PendingInsightGUID = ""
	s.AIStatus = "AI: generating summary and tags..."

	return tea.Batch(
		s.Spinner.Tick,
		GenerateInsightCmd(deps.Insights, item.GUID, buildInsightRequest(item)),
	)
}

func feedFetchStatusMessage(report usecase.FeedFetchReport) string {
	if report.Requested <= 1 {
		return ""
	}
	if report.TimedOut > 0 {
		if report.TimedOut == 1 {
			return "1 feed timed out"
		}
		return fmt.Sprintf("%d feeds timed out", report.TimedOut)
	}
	if report.Failed > 0 {
		if report.Failed == 1 {
			return "1 feed failed to load"
		}
		return fmt.Sprintf("%d feeds failed to load", report.Failed)
	}
	return ""
}

func applyHydratedItemToList(model *list.Model, item *reading.HistoryItem) {
	if model == nil || item == nil {
		return
	}
	for idx, listItem := range model.Items() {
		current, ok := listItem.(*presenter.Item)
		if !ok || current == nil || current.GUID != item.GUID {
			continue
		}
		current.RawTitle = item.Title
		current.Desc = item.Description
		current.Content = item.Content
		current.Published = item.Published
		current.Link = item.Link
		current.AISummary = item.AISummary
		current.AITags = append([]string(nil), item.AITags...)
		current.AIUpdatedAt = item.AIUpdatedAt
		current.Bookmarked = item.IsBookmarked
		current.Read = item.IsRead
		current.BodyHydrated = true
		model.SetItem(idx, current)
		return
	}
}

func selectedFeedItem(s *state.ModelState) (*presenter.Item, bool) {
	if s == nil {
		return nil, false
	}
	item, ok := s.FeedList.SelectedItem().(*presenter.Item)
	if !ok || item == nil {
		return nil, false
	}
	return item, true
}

func selectedActionableArticleItem(s *state.ModelState) (*presenter.Item, bool) {
	if s == nil {
		return nil, false
	}
	item, ok := s.ArticleList.SelectedItem().(*presenter.Item)
	if !ok || item == nil || item.IsSectionHeader() {
		return nil, false
	}
	return item, true
}

func subscriptionIndexByURL(feeds []string, targetURL string) (int, bool) {
	for index, feedURL := range feeds {
		if feedURL == targetURL {
			return index, true
		}
	}
	return 0, false
}

func enterNewsTopicView(s *state.ModelState, digestItem *presenter.Item) {
	if s == nil || digestItem == nil {
		return
	}

	s.NewsTopicDigestGUID = digestItem.GUID
	s.NewsTopicTitle = digestItem.RawTitle
	if s.NewsTopicTitle == "" {
		s.NewsTopicTitle = digestItem.TitleText
	}
	s.NewsTopicSummary = strings.TrimSpace(digestItem.Content)
	if s.NewsTopicSummary == "" {
		s.NewsTopicSummary = strings.TrimSpace(digestItem.Desc)
	}
	s.NewsTopicTags = append([]string(nil), digestItem.AITags...)

	presenter.ApplyRelatedArticleList(&s.ArticleList, s.History, digestItem.RelatedGUIDs)
	s.Session = state.NewsTopicView
}

func selectArticleItemByGUID(model *list.Model, guid string) {
	if model == nil || strings.TrimSpace(guid) == "" {
		return
	}
	for idx, listItem := range model.Items() {
		item, ok := listItem.(*presenter.Item)
		if !ok || item == nil {
			continue
		}
		if item.GUID == guid {
			model.Select(idx)
			return
		}
	}
}
