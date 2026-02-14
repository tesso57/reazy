// Package feed provides functionality to fetch and parse RSS/Atom feeds.
package feed

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/tesso57/reazy/internal/application/usecase"
	"github.com/tesso57/reazy/internal/domain/reading"
)

const feedAcceptHeader = "application/atom+xml, application/rss+xml, application/feed+json, application/xml;q=0.9, text/xml;q=0.8, */*;q=0.5"

type acceptTransport struct {
	base http.RoundTripper
}

func (t acceptTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	if clone.Header.Get("Accept") == "" {
		clone.Header.Set("Accept", feedAcceptHeader)
	}
	return base.RoundTrip(clone)
}

// ParserFunc is exposed for testing.
// It allows mocking the feed parsing logic.
var ParserFunc = defaultParser

func defaultParser(ctx context.Context, url string) (*gofeed.Feed, error) {
	fp := gofeed.NewParser()
	fp.UserAgent = "Reazy/1.0"
	fp.Client = &http.Client{Transport: acceptTransport{base: http.DefaultTransport}}
	return fp.ParseURLWithContext(url, ctx)
}

// Fetch parses a feed from the given URL.
func Fetch(url string) (*reading.Feed, error) {
	return FetchWithTimeout(url, 10*time.Second)
}

// FetchWithTimeout parses a feed from the given URL with timeout.
func FetchWithTimeout(url string, timeout time.Duration) (*reading.Feed, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return FetchWithContext(ctx, url)
}

// FetchWithContext parses a feed from the given URL with context.
func FetchWithContext(ctx context.Context, url string) (*reading.Feed, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("feed url is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := ParserFunc(ctx, url)
	if err != nil {
		return nil, err
	}

	f := new(reading.Feed{
		Title: parsed.Title,
		URL:   url,
		Items: make([]reading.Item, len(parsed.Items)),
	})

	for i, item := range parsed.Items {
		pub := item.Published
		if pub == "" {
			pub = item.Updated
		}
		var date time.Time
		if item.PublishedParsed != nil {
			date = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			date = *item.UpdatedParsed
		}

		f.Items[i] = reading.Item{
			GUID:        item.GUID,
			Title:       item.Title,
			Link:        item.Link,
			Published:   pub,
			Description: item.Description,
			Content:     item.Content,
			Date:        date,
			FeedTitle:   parsed.Title,
			FeedURL:     url,
		}
	}

	return f, nil
}

// FetchAll parses multiple feeds concurrently and aggregates items.
func FetchAll(urls []string, opt usecase.FeedFetchOptions) (*reading.Feed, usecase.FeedFetchReport, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allItems []reading.Item
	report := usecase.FeedFetchReport{Requested: len(urls)}

	batchCtx := context.Background()
	var batchCancel context.CancelFunc
	if opt.BatchTimeout > 0 {
		batchCtx, batchCancel = context.WithTimeout(batchCtx, opt.BatchTimeout)
		defer batchCancel()
	}

	for _, url := range urls {
		url := strings.TrimSpace(url)
		if url == "" {
			continue
		}
		wg.Go(func() {
			feedCtx := batchCtx
			var cancel context.CancelFunc
			if opt.PerFeedTimeout > 0 {
				feedCtx, cancel = context.WithTimeout(batchCtx, opt.PerFeedTimeout)
			}
			if cancel != nil {
				defer cancel()
			}

			f, err := FetchWithContext(feedCtx, url)
			mu.Lock()
			defer mu.Unlock()

			if err == nil && f != nil {
				report.Succeeded++
				allItems = append(allItems, f.Items...)
				return
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				report.TimedOut++
				return
			}
			report.Failed++
		})
	}
	wg.Wait()

	// Sort by Date descending
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Date.After(allItems[j].Date)
	})

	return new(reading.Feed{
		Title: "All Feeds",
		URL:   reading.AllFeedsURL,
		Items: allItems,
	}), report, nil
}

// Fetcher implements the usecase.FeedFetcher interface.
type Fetcher struct{}

// Fetch fetches a single feed.
func (Fetcher) Fetch(url string) (*reading.Feed, error) {
	return Fetch(url)
}

// FetchAll fetches and aggregates multiple feeds.
func (Fetcher) FetchAll(urls []string, opt usecase.FeedFetchOptions) (*reading.Feed, usecase.FeedFetchReport, error) {
	return FetchAll(urls, opt)
}
