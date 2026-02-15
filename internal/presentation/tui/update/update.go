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
	"github.com/tesso57/reazy/internal/domain/subscription"
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
	FeedGrouping  *usecase.FeedGroupingService
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

// FeedGroupingCompletedMsg is emitted after AI feed grouping is applied.
type FeedGroupingCompletedMsg struct {
	Feeds     []string
	Groups    []subscription.FeedGroup
	Ungrouped []string
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

// GenerateFeedGroupingCmd creates a command to group feeds by AI and persist config.
func GenerateFeedGroupingCmd(groupingSvc *usecase.FeedGroupingService, subscriptions *usecase.SubscriptionService, feeds []string) tea.Cmd {
	feedSnapshot := append([]string(nil), feeds...)
	return func() tea.Msg {
		if groupingSvc == nil {
			return FeedGroupingCompletedMsg{Err: fmt.Errorf("codex integration is disabled")}
		}
		if subscriptions == nil {
			return FeedGroupingCompletedMsg{Err: fmt.Errorf("subscription service is not configured")}
		}

		result, err := groupingSvc.Group(context.Background(), feedSnapshot)
		if err != nil {
			return FeedGroupingCompletedMsg{Err: err}
		}

		updatedFeeds, supported, err := subscriptions.ReplaceFeedGroups(result.Groups, result.Ungrouped)
		if err != nil {
			return FeedGroupingCompletedMsg{Err: err}
		}
		if !supported {
			return FeedGroupingCompletedMsg{Err: fmt.Errorf("feed grouping persistence is not supported")}
		}

		persistedGroups := result.Groups
		if groups, ok, err := subscriptions.ListGroups(); err != nil {
			return FeedGroupingCompletedMsg{Err: err}
		} else if ok {
			persistedGroups = groups
		}

		return FeedGroupingCompletedMsg{
			Feeds:     updatedFeeds,
			Groups:    persistedGroups,
			Ungrouped: result.Ungrouped,
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
	if handleSectionJump(s, msg) {
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

func handleSectionJump(s *state.ModelState, msg tea.KeyMsg) bool {
	if s == nil {
		return false
	}

	activeList, ok := activeListForFiltering(s)
	if !ok || activeList.FilterState() == list.Filtering {
		return false
	}

	switch msg.String() {
	case "J":
		return jumpSelectionToAdjacentSection(activeList, 1)
	case "K":
		return jumpSelectionToAdjacentSection(activeList, -1)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		sectionNumber := int(msg.String()[0] - '0')
		return jumpSelectionToSectionNumber(activeList, sectionNumber)
	case "0":
		return jumpSelectionToSectionNumber(activeList, 10)
	default:
		return false
	}
}

func jumpSelectionToAdjacentSection(model *list.Model, direction int) bool {
	if model == nil {
		return false
	}
	headers := sectionHeaderIndexes(model.Items())
	if len(headers) == 0 {
		return false
	}

	currentIndex := model.Index()
	currentHeader := -1
	for _, headerIndex := range headers {
		if headerIndex <= currentIndex {
			currentHeader = headerIndex
			continue
		}
		break
	}

	targetHeader := -1
	switch {
	case direction > 0:
		for _, headerIndex := range headers {
			if headerIndex > currentHeader {
				targetHeader = headerIndex
				break
			}
		}
	case direction < 0:
		if currentHeader < 0 {
			return true
		}
		for _, headerIndex := range headers {
			if headerIndex >= currentHeader {
				break
			}
			targetHeader = headerIndex
		}
	}
	if targetHeader < 0 {
		return true
	}

	selectFirstItemInSection(model, targetHeader)
	return true
}

func jumpSelectionToSectionNumber(model *list.Model, sectionNumber int) bool {
	if model == nil {
		return false
	}
	headers := sectionHeaderIndexes(model.Items())
	if len(headers) == 0 {
		return false
	}
	if sectionNumber <= 0 || sectionNumber > len(headers) {
		return true
	}

	selectFirstItemInSection(model, headers[sectionNumber-1])
	return true
}

func sectionHeaderIndexes(items []list.Item) []int {
	headers := make([]int, 0, len(items))
	for index, item := range items {
		feedItem, ok := item.(*presenter.Item)
		if !ok || feedItem == nil || !feedItem.IsSectionHeader() {
			continue
		}
		headers = append(headers, index)
	}
	return headers
}

func selectFirstItemInSection(model *list.Model, headerIndex int) {
	if model == nil || headerIndex < 0 || headerIndex >= len(model.Items()) {
		return
	}

	items := model.Items()
	targetIndex := headerIndex
	for index := headerIndex + 1; index < len(items); index++ {
		feedItem, ok := items[index].(*presenter.Item)
		if !ok || feedItem == nil {
			continue
		}
		if feedItem.IsSectionHeader() {
			break
		}
		targetIndex = index
		break
	}
	model.Select(targetIndex)
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

// HandleFeedGroupingCompletedMsg applies grouped feeds to state.
func HandleFeedGroupingCompletedMsg(s *state.ModelState, msg FeedGroupingCompletedMsg) {
	s.Loading = false
	s.AIStatus = ""
	defer UpdateListSizes(s)

	if msg.Err != nil {
		s.Err = msg.Err
		s.StatusMessage = fmt.Sprintf("AI feed grouping failed: %s", strings.TrimSpace(msg.Err.Error()))
		return
	}

	s.Err = nil
	s.Feeds = append([]string(nil), msg.Feeds...)
	s.FeedGroups = cloneFeedGroups(msg.Groups)
	presenter.ApplyFeedList(&s.FeedList, s.Feeds, s.FeedGroups)

	groupedCount := len(msg.Feeds) - len(msg.Ungrouped)
	if groupedCount < 0 {
		groupedCount = 0
	}
	s.StatusMessage = fmt.Sprintf("AI grouped %d feeds into %d groups", groupedCount, len(msg.Groups))
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
				syncFeedGroupsFromRepository(s, deps)
				presenter.ApplyFeedList(&s.FeedList, s.Feeds, s.FeedGroups)
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
			if item.SubscriptionIndex >= 0 && item.SubscriptionIndex < len(s.Feeds) {
				feeds, err := deps.Subscriptions.Remove(item.SubscriptionIndex)
				if err != nil {
					s.Err = err
				} else {
					s.Feeds = feeds
					if !syncFeedGroupsFromRepository(s, deps) {
						removeFeedFromGroupState(s, item.GroupName, item.Link)
					}
					presenter.ApplyFeedList(&s.FeedList, s.Feeds, s.FeedGroups)
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
			if i.IsSectionHeader() {
				return nil, true
			}
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
	case intent.GroupFeeds, intent.Summarize:
		return startFeedGrouping(s, deps), true
	case intent.ToggleHelp:
		s.Help.ShowAll = !s.Help.ShowAll
		return nil, true
	}
	return nil, false
}

func startFeedGrouping(s *state.ModelState, deps Deps) tea.Cmd {
	s.Loading = true
	s.Err = nil
	s.StatusMessage = ""
	s.AIStatus = "AI: grouping feeds..."
	return tea.Batch(s.Spinner.Tick, GenerateFeedGroupingCmd(deps.FeedGrouping, deps.Subscriptions, s.Feeds))
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
	if !ok || item == nil || item.IsSectionHeader() {
		return nil, false
	}
	return item, true
}

func syncFeedGroupsFromRepository(s *state.ModelState, deps Deps) bool {
	if s == nil || deps.Subscriptions == nil {
		return false
	}
	groups, supported, err := deps.Subscriptions.ListGroups()
	if err != nil {
		s.Err = err
		return false
	}
	if !supported {
		return false
	}
	s.FeedGroups = groups
	return true
}

func removeFeedFromGroupState(s *state.ModelState, groupName, targetURL string) {
	if s == nil || targetURL == "" {
		return
	}
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return
	}

	for groupIndex := range s.FeedGroups {
		if s.FeedGroups[groupIndex].Name != groupName {
			continue
		}
		feeds := s.FeedGroups[groupIndex].Feeds
		for feedIndex, feedURL := range feeds {
			if feedURL != targetURL {
				continue
			}
			s.FeedGroups[groupIndex].Feeds = append(feeds[:feedIndex], feeds[feedIndex+1:]...)
			if len(s.FeedGroups[groupIndex].Feeds) == 0 {
				s.FeedGroups = append(s.FeedGroups[:groupIndex], s.FeedGroups[groupIndex+1:]...)
			}
			return
		}
	}
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
