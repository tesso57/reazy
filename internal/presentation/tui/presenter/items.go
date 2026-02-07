// Package presenter builds view models for the TUI.
package presenter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/tesso57/reazy/internal/domain/reading"
	"github.com/tesso57/reazy/internal/presentation/tui/textutil"
)

// Item is a view model for list items.
type Item struct {
	TitleText     string
	RawTitle      string
	Desc          string
	Content       string
	Link          string
	Published     string
	GUID          string
	Read          bool
	Bookmarked    bool
	AISummary     string
	AITags        []string
	AIUpdatedAt   time.Time
	FeedTitleText string
	FeedURL       string
}

// FilterValue implements list.Item.
func (i *Item) FilterValue() string { return i.TitleText }

// Title returns the item title.
func (i *Item) Title() string { return i.TitleText }

// URL returns the item's URL.
func (i *Item) URL() string { return i.Link }

// IsRead returns the read state.
func (i *Item) IsRead() bool { return i.Read }

// IsBookmarked returns the bookmarked state.
func (i *Item) IsBookmarked() bool { return i.Bookmarked }

// HasAISummary returns true when AI summary is available.
func (i *Item) HasAISummary() bool { return strings.TrimSpace(i.AISummary) != "" }

// FeedTitle returns the feed title for the item.
func (i *Item) FeedTitle() string { return i.FeedTitleText }

// Description returns a formatted description for list display.
func (i *Item) Description() string {
	if i.Published != "" {
		return fmt.Sprintf("%s - %s", i.Published, i.Desc)
	}
	return i.Desc
}

// BuildFeedListItems builds list items for the feed list.
func BuildFeedListItems(feeds []string) []list.Item {
	items := make([]list.Item, len(feeds)+2)
	items[0] = &Item{TitleText: "0. * All Feeds", RawTitle: "All Feeds", Link: reading.AllFeedsURL}
	items[1] = &Item{TitleText: "1. * Bookmarks", RawTitle: "Bookmarks", Link: reading.BookmarksURL}

	for i, f := range feeds {
		items[i+2] = &Item{TitleText: fmt.Sprintf("%d. %s", i+2, textutil.SingleLine(f)), RawTitle: f, Link: f}
	}
	return items
}

// ApplyFeedList updates the list model with feed items.
func ApplyFeedList(model *list.Model, feeds []string) {
	model.SetItems(BuildFeedListItems(feeds))
}

// BuildArticleListItems builds list items for articles.
func BuildArticleListItems(history *reading.History, feedURL string) []list.Item {
	if history == nil {
		return nil
	}
	items := history.ItemsByFeed(feedURL)

	sort.Slice(items, func(i, j int) bool {
		return items[i].Date.After(items[j].Date)
	})

	result := make([]list.Item, len(items))
	for i, it := range items {
		title := textutil.SingleLine(it.Title)
		feedTitle := textutil.SingleLine(it.FeedTitle)
		if (feedURL == reading.AllFeedsURL || feedURL == reading.BookmarksURL) && feedTitle != "" {
			title = fmt.Sprintf("%d. [%s] %s", i+1, feedTitle, title)
		} else {
			title = fmt.Sprintf("%d. %s", i+1, title)
		}

		result[i] = &Item{
			TitleText:     title,
			RawTitle:      it.Title,
			Desc:          it.Description,
			Content:       it.Content,
			Link:          it.Link,
			Published:     it.Published,
			GUID:          it.GUID,
			Read:          it.IsRead,
			Bookmarked:    it.IsBookmarked,
			AISummary:     it.AISummary,
			AITags:        append([]string(nil), it.AITags...),
			AIUpdatedAt:   it.AIUpdatedAt,
			FeedTitleText: it.FeedTitle,
			FeedURL:       it.FeedURL,
		}
	}
	return result
}

// ApplyArticleList updates the article list and title based on feed URL.
func ApplyArticleList(model *list.Model, history *reading.History, feedURL string) {
	model.SetItems(BuildArticleListItems(history, feedURL))
	model.SetItems(BuildArticleListItems(history, feedURL))
	if feedURL == reading.AllFeedsURL {
		model.Title = "All Feeds"
	} else if feedURL == reading.BookmarksURL {
		model.Title = "Bookmarks"
	} else {
		model.Title = "Articles"
	}
}
