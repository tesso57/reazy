package feed

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type Item struct {
	Title       string
	Link        string
	Published   string
	Description string
	Content     string
	Date        time.Time
	FeedTitle   string
	FeedURL     string
}

type Feed struct {
	Title string
	Items []Item
	URL   string
}

// ParserFunc is exposed for testing
var ParserFunc = func(url string) (*gofeed.Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	return fp.ParseURLWithContext(url, ctx)
}

func Fetch(url string) (*Feed, error) {
	parsed, err := ParserFunc(url)
	if err != nil {
		return nil, err
	}

	f := &Feed{
		Title: parsed.Title,
		URL:   url,
		Items: make([]Item, len(parsed.Items)),
	}

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

		f.Items[i] = Item{
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

func FetchAll(urls []string) (*Feed, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allItems []Item

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			f, err := Fetch(u)
			if err == nil {
				mu.Lock()
				allItems = append(allItems, f.Items...)
				mu.Unlock()
			}
		}(url)
	}
	wg.Wait()

	// Sort by Date descending
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Date.After(allItems[j].Date)
	})

	return &Feed{
		Title: "All Feeds",
		URL:   "internal://all", // Special URL
		Items: allItems,
	}, nil
}
