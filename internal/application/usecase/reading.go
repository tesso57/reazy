// Package usecase contains application-level services.
package usecase

import (
	"time"

	"github.com/tesso57/reazy/internal/domain/reading"
)

// FeedFetcher abstracts RSS fetching.
type FeedFetcher interface {
	Fetch(url string) (*reading.Feed, error)
	FetchAll(urls []string) (*reading.Feed, error)
}

// HistoryRepository abstracts history persistence.
type HistoryRepository interface {
	Load() (map[string]*reading.HistoryItem, error)
	Save(items []*reading.HistoryItem) error
}

// ReadingService coordinates feed fetching and history persistence.
type ReadingService struct {
	Fetcher     FeedFetcher
	HistoryRepo HistoryRepository
	Now         func() time.Time
}

// NewReadingService constructs a ReadingService.
func NewReadingService(fetcher FeedFetcher, historyRepo HistoryRepository, now func() time.Time) ReadingService {
	return ReadingService{
		Fetcher:     fetcher,
		HistoryRepo: historyRepo,
		Now:         now,
	}
}

// FetchFeed fetches a single feed or the aggregated "All Feeds".
func (s ReadingService) FetchFeed(url string, all []string) (*reading.Feed, error) {
	if url == reading.AllFeedsURL {
		return s.Fetcher.FetchAll(all)
	}
	if url == reading.BookmarksURL {
		// Bookmarks are local, no fetch needed. Return empty feed or nil.
		return &reading.Feed{
			Title: "Bookmarks",
			URL:   reading.BookmarksURL,
			Items: []reading.Item{},
		}, nil
	}
	return s.Fetcher.Fetch(url)
}

// LoadHistory loads history from persistence.
func (s ReadingService) LoadHistory() (*reading.History, error) {
	items, err := s.HistoryRepo.Load()
	return reading.NewHistory(items), err
}

// SaveHistory persists the current history snapshot.
func (s ReadingService) SaveHistory(history *reading.History) error {
	if history == nil {
		return nil
	}
	return s.HistoryRepo.Save(history.Snapshot())
}

// MergeHistory merges fetched feed items into history.
func (s ReadingService) MergeHistory(history *reading.History, feed *reading.Feed) {
	if history == nil {
		return
	}
	history.MergeFeed(feed, s.now())
}

// ToggleBookmark toggles the bookmark status of an item and persists the change.
func (s ReadingService) ToggleBookmark(history *reading.History, guid string) error {
	if history == nil {
		return nil
	}
	if history.ToggleBookmark(guid) {
		return s.SaveHistory(history)
	}
	return nil
}

func (s ReadingService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
