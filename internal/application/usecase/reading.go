// Package usecase contains application-level services.
package usecase

import (
	"strings"
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

// FeedFetchOptions controls multi-feed fetch behavior.
type FeedFetchOptions struct {
	PerFeedTimeout time.Duration
	BatchTimeout   time.Duration
}

// FeedFetchReport represents aggregate results of multi-feed fetching.
type FeedFetchReport struct {
	Requested int
	Succeeded int
	Failed    int
	TimedOut  int
}

var defaultFeedFetchOptions = FeedFetchOptions{
	PerFeedTimeout: 8 * time.Second,
	BatchTimeout:   12 * time.Second,
}

// FeedFetcher abstracts RSS fetching.
type FeedFetcher interface {
	Fetch(url string) (*reading.Feed, error)
	FetchAll(urls []string, opt FeedFetchOptions) (*reading.Feed, FeedFetchReport, error)
}

// HistoryRepository abstracts history persistence.
type HistoryRepository interface {
	LoadMetadata() (map[string]*reading.HistoryItem, error)
	LoadByGUID(guid string) (*reading.HistoryItem, error)
	Upsert(items []*reading.HistoryItem) error
	SetRead(guid string, isRead bool) error
	SetBookmark(guid string, isBookmarked bool) error
	SetInsight(guid, summary string, tags []string, updatedAt time.Time) error
	ReplaceDigestItemsByDate(dateKey string, items []*reading.HistoryItem) error
	LoadTodayArticles(dateKey string, feeds []string, limit int, loc *time.Location) ([]*reading.HistoryItem, error)
}

// ReadingService coordinates feed fetching and history persistence.
type ReadingService struct {
	Fetcher     FeedFetcher
	HistoryRepo HistoryRepository
	Now         func() time.Time
}

// NewReadingService constructs a ReadingService.
func NewReadingService(fetcher FeedFetcher, historyRepo HistoryRepository, now func() time.Time) *ReadingService {
	return new(ReadingService{
		Fetcher:     fetcher,
		HistoryRepo: historyRepo,
		Now:         now,
	})
}

// FetchFeed fetches a single feed or a virtual aggregated feed.
func (s *ReadingService) FetchFeed(url string, all []string) (*reading.Feed, FeedFetchReport, error) {
	if url == reading.AllFeedsURL {
		return s.Fetcher.FetchAll(all, defaultFeedFetchOptions)
	}
	if url == reading.NewsURL {
		feed, report, err := s.Fetcher.FetchAll(all, defaultFeedFetchOptions)
		if feed != nil {
			feed.URL = reading.NewsURL
			if feed.Title == "" {
				feed.Title = "News"
			}
		}
		return feed, report, err
	}
	if url == reading.BookmarksURL {
		// Bookmarks are local, no fetch needed. Return empty feed or nil.
		return new(reading.Feed{
			Title: "Bookmarks",
			URL:   reading.BookmarksURL,
			Items: []reading.Item{},
		}), FeedFetchReport{}, nil
	}
	feed, err := s.Fetcher.Fetch(url)
	report := FeedFetchReport{Requested: 1}
	if err != nil {
		report.Failed = 1
	} else {
		report.Succeeded = 1
	}
	return feed, report, err
}

// LoadHistoryMetadata loads history metadata from persistence.
func (s *ReadingService) LoadHistoryMetadata() (*reading.History, error) {
	if s.HistoryRepo == nil {
		return reading.NewHistory(nil), nil
	}
	items, err := s.HistoryRepo.LoadMetadata()
	return reading.NewHistory(items), err
}

// LoadHistoryItem loads one fully-hydrated history item by GUID.
func (s *ReadingService) LoadHistoryItem(guid string) (*reading.HistoryItem, error) {
	if s.HistoryRepo == nil || strings.TrimSpace(guid) == "" {
		return nil, nil
	}
	return s.HistoryRepo.LoadByGUID(guid)
}

// LoadTodayArticles loads today's source articles for digest generation.
func (s *ReadingService) LoadTodayArticles(dateKey string, feeds []string, limit int, loc *time.Location) ([]*reading.HistoryItem, error) {
	if s.HistoryRepo == nil {
		return nil, nil
	}
	return s.HistoryRepo.LoadTodayArticles(dateKey, feeds, limit, loc)
}

// MergeHistory merges fetched feed items into history and persists updated items.
func (s *ReadingService) MergeHistory(history *reading.History, feed *reading.Feed) error {
	if history == nil {
		return nil
	}
	changed := history.MergeFeed(feed, s.now())
	if len(changed) == 0 || s.HistoryRepo == nil {
		return nil
	}
	return s.HistoryRepo.Upsert(changed)
}

// MarkRead marks an article as read and persists the change.
func (s *ReadingService) MarkRead(history *reading.History, guid string) error {
	if history == nil || strings.TrimSpace(guid) == "" {
		return nil
	}
	if !history.MarkRead(guid) {
		return nil
	}
	if s.HistoryRepo == nil {
		return nil
	}
	return s.HistoryRepo.SetRead(guid, true)
}

// ToggleBookmark toggles the bookmark status of an item and persists the change.
func (s *ReadingService) ToggleBookmark(history *reading.History, guid string) error {
	if history == nil || strings.TrimSpace(guid) == "" {
		return nil
	}
	if history.ToggleBookmark(guid) {
		if s.HistoryRepo == nil {
			return nil
		}
		item, ok := history.Item(guid)
		if !ok || item == nil {
			return nil
		}
		return s.HistoryRepo.SetBookmark(guid, item.IsBookmarked)
	}
	return nil
}

// ApplyInsight applies AI-generated insight and persists only updated fields.
func (s *ReadingService) ApplyInsight(history *reading.History, guid string, insight Insight) (time.Time, bool, error) {
	updatedAt := s.now()
	if history == nil || strings.TrimSpace(guid) == "" {
		return updatedAt, false, nil
	}
	if !history.SetInsight(guid, insight.Summary, insight.Tags, updatedAt) {
		return updatedAt, false, nil
	}
	if s.HistoryRepo == nil {
		return updatedAt, true, nil
	}
	if err := s.HistoryRepo.SetInsight(guid, insight.Summary, insight.Tags, updatedAt); err != nil {
		return updatedAt, true, err
	}
	return updatedAt, true, nil
}

// ReplaceDigestItemsByDate updates digest items in memory and persistence.
func (s *ReadingService) ReplaceDigestItemsByDate(history *reading.History, dateKey string, items []*reading.HistoryItem) error {
	if history == nil || strings.TrimSpace(dateKey) == "" {
		return nil
	}
	history.ReplaceDigestItemsByDate(dateKey, items)
	if s.HistoryRepo == nil {
		return nil
	}
	return s.HistoryRepo.ReplaceDigestItemsByDate(dateKey, items)
}

func (s *ReadingService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
