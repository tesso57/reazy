package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tesso57/reazy/internal/domain/reading"
)

type mockHistoryRepo struct {
	mock.Mock
}

func (m *mockHistoryRepo) LoadMetadata() (map[string]*reading.HistoryItem, error) {
	args := m.Called()
	items, _ := args.Get(0).(map[string]*reading.HistoryItem)
	return items, args.Error(1)
}

func (m *mockHistoryRepo) LoadByGUID(guid string) (*reading.HistoryItem, error) {
	args := m.Called(guid)
	item, _ := args.Get(0).(*reading.HistoryItem)
	return item, args.Error(1)
}

func (m *mockHistoryRepo) Upsert(items []*reading.HistoryItem) error {
	args := m.Called(items)
	return args.Error(0)
}

func (m *mockHistoryRepo) SetRead(guid string, isRead bool) error {
	args := m.Called(guid, isRead)
	return args.Error(0)
}

func (m *mockHistoryRepo) SetBookmark(guid string, isBookmarked bool) error {
	args := m.Called(guid, isBookmarked)
	return args.Error(0)
}

func (m *mockHistoryRepo) SetInsight(guid, summary string, tags []string, updatedAt time.Time) error {
	args := m.Called(guid, summary, tags, updatedAt)
	return args.Error(0)
}

func (m *mockHistoryRepo) ReplaceDigestItemsByDate(dateKey string, items []*reading.HistoryItem) error {
	args := m.Called(dateKey, items)
	return args.Error(0)
}

func (m *mockHistoryRepo) LoadTodayArticles(dateKey string, feeds []string, limit int, loc *time.Location) ([]*reading.HistoryItem, error) {
	args := m.Called(dateKey, feeds, limit, loc)
	items, _ := args.Get(0).([]*reading.HistoryItem)
	return items, args.Error(1)
}

func TestReadingService_ToggleBookmark(t *testing.T) {
	tests := []struct {
		name      string
		repoErr   error
		guid      string
		initial   map[string]*reading.HistoryItem
		wantCall  bool
		wantValue bool
		wantErr   bool
	}{
		{
			name:    "nil history",
			guid:    "1",
			wantErr: false,
		},
		{
			name: "item not found",
			initial: map[string]*reading.HistoryItem{
				"2": {GUID: "2"},
			},
			guid:    "1",
			wantErr: false,
		},
		{
			name: "success toggle",
			initial: map[string]*reading.HistoryItem{
				"1": {GUID: "1", IsBookmarked: false},
			},
			guid:      "1",
			wantCall:  true,
			wantValue: true,
			wantErr:   false,
		},
		{
			name:    "repo error",
			repoErr: assertErr("save failed"),
			initial: map[string]*reading.HistoryItem{
				"1": {GUID: "1", IsBookmarked: false},
			},
			guid:      "1",
			wantCall:  true,
			wantValue: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockHistoryRepo{}
			svc := NewReadingService(nil, repo, nil)

			var history *reading.History
			if tt.initial != nil {
				history = reading.NewHistory(tt.initial)
			}

			if tt.wantCall {
				repo.On("SetBookmark", tt.guid, tt.wantValue).Return(tt.repoErr).Once()
			}

			err := svc.ToggleBookmark(history, tt.guid)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ToggleBookmark() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantCall {
				repo.AssertExpectations(t)
			} else {
				repo.AssertNotCalled(t, "SetBookmark", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestReadingService_MarkRead(t *testing.T) {
	repo := &mockHistoryRepo{}
	svc := NewReadingService(nil, repo, nil)
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"1": {GUID: "1"},
	})

	repo.On("SetRead", "1", true).Return(nil).Once()
	if err := svc.MarkRead(history, "1"); err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
	repo.AssertExpectations(t)
}

func TestReadingService_ApplyInsight(t *testing.T) {
	now := time.Date(2026, 2, 14, 9, 30, 0, 0, time.UTC)
	repo := &mockHistoryRepo{}
	svc := NewReadingService(nil, repo, func() time.Time { return now })
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"1": {GUID: "1", Kind: reading.ArticleKind},
	})

	repo.On("SetInsight", "1", "s", []string{"go"}, now).Return(nil).Once()

	updatedAt, ok, err := svc.ApplyInsight(history, "1", Insight{Summary: "s", Tags: []string{"go"}})
	if err != nil {
		t.Fatalf("ApplyInsight() error = %v", err)
	}
	if !ok {
		t.Fatal("ApplyInsight() should succeed")
	}
	if !updatedAt.Equal(now) {
		t.Fatalf("updatedAt = %v, want %v", updatedAt, now)
	}
	repo.AssertExpectations(t)
}

type mockFeedFetcher struct {
	mock.Mock
}

func (m *mockFeedFetcher) Fetch(url string) (*reading.Feed, error) {
	args := m.Called(url)
	feed, _ := args.Get(0).(*reading.Feed)
	return feed, args.Error(1)
}

func (m *mockFeedFetcher) FetchAll(urls []string, opt FeedFetchOptions) (*reading.Feed, FeedFetchReport, error) {
	args := m.Called(urls, opt)
	feed, _ := args.Get(0).(*reading.Feed)
	report, _ := args.Get(1).(FeedFetchReport)
	return feed, report, args.Error(2)
}

func TestReadingService_FetchFeed_NewsUsesFetchAll(t *testing.T) {
	fetcher := &mockFeedFetcher{}
	svc := NewReadingService(fetcher, nil, nil)

	all := []string{"https://example.com/rss", "https://example.com/atom"}
	feedFromFetcher := &reading.Feed{Title: "All Feeds", URL: reading.AllFeedsURL}
	reportFromFetcher := FeedFetchReport{Requested: 2, Succeeded: 2}

	fetcher.On("FetchAll", all, mock.MatchedBy(func(opt FeedFetchOptions) bool {
		return opt.PerFeedTimeout > 0 && opt.BatchTimeout > 0
	})).Return(feedFromFetcher, reportFromFetcher, nil).Once()

	feed, report, err := svc.FetchFeed(reading.NewsURL, all)
	if err != nil {
		t.Fatalf("FetchFeed(news) error = %v", err)
	}
	if feed == nil {
		t.Fatal("FetchFeed(news) should return feed")
	}
	if feed.URL != reading.NewsURL {
		t.Fatalf("feed.URL = %q, want %q", feed.URL, reading.NewsURL)
	}
	if report.Requested != 2 || report.Succeeded != 2 {
		t.Fatalf("report = %+v", report)
	}
	fetcher.AssertExpectations(t)
}

func TestReadingService_FetchFeed_BookmarksSkipsFetcher(t *testing.T) {
	fetcher := &mockFeedFetcher{}
	svc := NewReadingService(fetcher, nil, nil)

	feed, report, err := svc.FetchFeed(reading.BookmarksURL, []string{"https://example.com/rss"})
	if err != nil {
		t.Fatalf("FetchFeed(bookmarks) error = %v", err)
	}
	if feed == nil {
		t.Fatal("FetchFeed(bookmarks) should return feed")
	}
	if feed.URL != reading.BookmarksURL {
		t.Fatalf("feed.URL = %q, want %q", feed.URL, reading.BookmarksURL)
	}
	if report.Requested != 0 {
		t.Fatalf("bookmarks report should be empty, got %+v", report)
	}
	fetcher.AssertNotCalled(t, "Fetch", mock.Anything)
	fetcher.AssertNotCalled(t, "FetchAll", mock.Anything, mock.Anything)
}

type errValue string

func (e errValue) Error() string { return string(e) }

func assertErr(msg string) error { return errValue(msg) }
