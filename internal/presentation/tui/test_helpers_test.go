package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/application/usecase"
	"github.com/tesso57/reazy/internal/domain/reading"
)

type stubSubscriptionRepo struct {
	mock.Mock
	feeds []string
}

func (s *stubSubscriptionRepo) List() ([]string, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		feeds, _ := args.Get(0).([]string)
		return feeds, args.Error(1)
	}
	feeds := make([]string, len(s.feeds))
	copy(feeds, s.feeds)
	return feeds, nil
}

func (s *stubSubscriptionRepo) Add(url string) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(url)
		return args.Error(0)
	}
	s.feeds = append(s.feeds, url)
	return nil
}

func (s *stubSubscriptionRepo) Remove(index int) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(index)
		return args.Error(0)
	}
	if index < 0 || index >= len(s.feeds) {
		return fmt.Errorf("invalid feed index: %d", index)
	}
	s.feeds = append(s.feeds[:index], s.feeds[index+1:]...)
	return nil
}

type stubHistoryRepo struct {
	mock.Mock
	items map[string]*reading.HistoryItem
}

func (s *stubHistoryRepo) LoadMetadata() (map[string]*reading.HistoryItem, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		items, _ := args.Get(0).(map[string]*reading.HistoryItem)
		return items, args.Error(1)
	}
	if s.items == nil {
		s.items = make(map[string]*reading.HistoryItem)
	}
	return s.items, nil
}

func (s *stubHistoryRepo) LoadByGUID(guid string) (*reading.HistoryItem, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(guid)
		item, _ := args.Get(0).(*reading.HistoryItem)
		return item, args.Error(1)
	}
	if s.items == nil {
		return nil, nil
	}
	item, ok := s.items[guid]
	if !ok {
		return nil, nil
	}
	copyItem := *item
	copyItem.BodyHydrated = true
	return &copyItem, nil
}

func (s *stubHistoryRepo) Upsert(items []*reading.HistoryItem) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(items)
		return args.Error(0)
	}
	if s.items == nil {
		s.items = make(map[string]*reading.HistoryItem)
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		s.items[item.GUID] = item
	}
	return nil
}

func (s *stubHistoryRepo) SetRead(guid string, isRead bool) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(guid, isRead)
		return args.Error(0)
	}
	if item, ok := s.items[guid]; ok && item != nil {
		item.IsRead = isRead
	}
	return nil
}

func (s *stubHistoryRepo) SetBookmark(guid string, isBookmarked bool) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(guid, isBookmarked)
		return args.Error(0)
	}
	if item, ok := s.items[guid]; ok && item != nil {
		item.IsBookmarked = isBookmarked
	}
	return nil
}

func (s *stubHistoryRepo) SetInsight(guid, summary string, tags []string, updatedAt time.Time) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(guid, summary, tags, updatedAt)
		return args.Error(0)
	}
	if item, ok := s.items[guid]; ok && item != nil {
		item.AISummary = summary
		item.AITags = append([]string(nil), tags...)
		item.AIUpdatedAt = updatedAt
	}
	return nil
}

func (s *stubHistoryRepo) ReplaceDigestItemsByDate(dateKey string, items []*reading.HistoryItem) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(dateKey, items)
		return args.Error(0)
	}
	if s.items == nil {
		s.items = make(map[string]*reading.HistoryItem)
	}
	for _, item := range items {
		if item == nil || item.GUID == "" {
			continue
		}
		item.Kind = reading.NewsDigestKind
		item.DigestDate = dateKey
		s.items[item.GUID] = item
	}
	return nil
}

func (s *stubHistoryRepo) LoadTodayArticles(dateKey string, feeds []string, limit int, loc *time.Location) ([]*reading.HistoryItem, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(dateKey, feeds, limit, loc)
		items, _ := args.Get(0).([]*reading.HistoryItem)
		return items, args.Error(1)
	}
	if s.items == nil {
		return nil, nil
	}
	history := reading.NewHistory(s.items)
	result := history.TodayArticleItems(dateKey, feeds, loc)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	out := make([]*reading.HistoryItem, 0, len(result))
	for _, item := range result {
		if item == nil {
			continue
		}
		copyItem := *item
		copyItem.BodyHydrated = true
		out = append(out, &copyItem)
	}
	return out, nil
}

type stubFeedFetcher struct {
	mock.Mock
	feed *reading.Feed
	err  error
}

func (s *stubFeedFetcher) Fetch(_ string) (*reading.Feed, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		feed, _ := args.Get(0).(*reading.Feed)
		return feed, args.Error(1)
	}
	return s.feed, s.err
}

func (s *stubFeedFetcher) FetchAll(_ []string, _ usecase.FeedFetchOptions) (*reading.Feed, usecase.FeedFetchReport, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		feed, _ := args.Get(0).(*reading.Feed)
		report, _ := args.Get(1).(usecase.FeedFetchReport)
		return feed, report, args.Error(2)
	}
	return s.feed, usecase.FeedFetchReport{}, s.err
}

type stubInsightGenerator struct {
	mock.Mock
	insight usecase.Insight
	err     error
}

func (s *stubInsightGenerator) Generate(_ context.Context, _ usecase.InsightRequest) (usecase.Insight, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		insight, _ := args.Get(0).(usecase.Insight)
		return insight, args.Error(1)
	}
	return s.insight, s.err
}

type stubNewsDigestGenerator struct {
	mock.Mock
	topics []usecase.NewsDigestTopic
	err    error
}

func (s *stubNewsDigestGenerator) Generate(_ context.Context, _ usecase.NewsDigestRequest) ([]usecase.NewsDigestTopic, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		topics, _ := args.Get(0).([]usecase.NewsDigestTopic)
		return topics, args.Error(1)
	}
	return s.topics, s.err
}

func newTestModel(cfg settings.Settings, subsRepo usecase.SubscriptionRepository, historyRepo usecase.HistoryRepository, fetcher usecase.FeedFetcher) *Model {
	subs := usecase.NewSubscriptionService(subsRepo)
	readingSvc := usecase.NewReadingService(fetcher, historyRepo, nil)
	return NewModel(cfg, subs, readingSvc)
}

func newTestModelWithInsightGenerator(
	cfg settings.Settings,
	subsRepo usecase.SubscriptionRepository,
	historyRepo usecase.HistoryRepository,
	fetcher usecase.FeedFetcher,
	insightGen usecase.InsightGenerator,
) *Model {
	return newTestModelWithInsightAndNewsDigestGenerator(cfg, subsRepo, historyRepo, fetcher, insightGen, nil)
}

func newTestModelWithInsightAndNewsDigestGenerator(
	cfg settings.Settings,
	subsRepo usecase.SubscriptionRepository,
	historyRepo usecase.HistoryRepository,
	fetcher usecase.FeedFetcher,
	insightGen usecase.InsightGenerator,
	newsDigestGen usecase.NewsDigestGenerator,
) *Model {
	subs := usecase.NewSubscriptionService(subsRepo)
	readingSvc := usecase.NewReadingService(fetcher, historyRepo, nil)
	insightSvc := usecase.NewInsightService(insightGen, nil)
	newsSvc := usecase.NewNewsDigestService(newsDigestGen, nil, nil)
	return NewModelWithServices(cfg, subs, readingSvc, insightSvc, newsSvc)
}
