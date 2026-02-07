package tui

import (
	"context"
	"fmt"

	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/application/usecase"
	"github.com/tesso57/reazy/internal/domain/reading"
)

type stubSubscriptionRepo struct {
	feeds []string
}

func (s *stubSubscriptionRepo) List() ([]string, error) {
	feeds := make([]string, len(s.feeds))
	copy(feeds, s.feeds)
	return feeds, nil
}

func (s *stubSubscriptionRepo) Add(url string) error {
	s.feeds = append(s.feeds, url)
	return nil
}

func (s *stubSubscriptionRepo) Remove(index int) error {
	if index < 0 || index >= len(s.feeds) {
		return fmt.Errorf("invalid feed index: %d", index)
	}
	s.feeds = append(s.feeds[:index], s.feeds[index+1:]...)
	return nil
}

type stubHistoryRepo struct {
	items map[string]*reading.HistoryItem
}

func (s *stubHistoryRepo) Load() (map[string]*reading.HistoryItem, error) {
	if s.items == nil {
		s.items = make(map[string]*reading.HistoryItem)
	}
	return s.items, nil
}

func (s *stubHistoryRepo) Save(items []*reading.HistoryItem) error {
	if s.items == nil {
		s.items = make(map[string]*reading.HistoryItem)
	}
	for k := range s.items {
		delete(s.items, k)
	}
	for _, item := range items {
		s.items[item.GUID] = item
	}
	return nil
}

type stubFeedFetcher struct {
	feed *reading.Feed
	err  error
}

func (s stubFeedFetcher) Fetch(_ string) (*reading.Feed, error) {
	return s.feed, s.err
}

func (s stubFeedFetcher) FetchAll(_ []string) (*reading.Feed, error) {
	return s.feed, s.err
}

type stubInsightGenerator struct {
	insight usecase.Insight
	err     error
}

func (s stubInsightGenerator) Generate(_ context.Context, _ usecase.InsightRequest) (usecase.Insight, error) {
	return s.insight, s.err
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
	subs := usecase.NewSubscriptionService(subsRepo)
	readingSvc := usecase.NewReadingService(fetcher, historyRepo, nil)
	insightSvc := usecase.NewInsightService(insightGen, nil)
	return NewModelWithInsights(cfg, subs, readingSvc, insightSvc)
}
