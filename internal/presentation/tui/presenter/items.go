// Package presenter builds view models for the TUI.
package presenter

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	"github.com/tesso57/reazy/internal/domain/reading"
)

// Item is a view model for list items.
type Item struct {
	TitleText     string
	Desc          string
	Content       string
	Link          string
	Published     string
	GUID          string
	Read          bool
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
	items := make([]list.Item, len(feeds)+1)
	items[0] = &Item{TitleText: "0. * All Feeds", Link: reading.AllFeedsURL}
	for i, f := range feeds {
		items[i+1] = &Item{TitleText: fmt.Sprintf("%d. %s", i+1, f), Link: f}
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
		title := it.Title
		if feedURL == reading.AllFeedsURL && it.FeedTitle != "" {
			title = fmt.Sprintf("[%s] %s", it.FeedTitle, title)
		}

		result[i] = &Item{
			TitleText:     title,
			Desc:          it.Description,
			Content:       it.Content,
			Link:          it.Link,
			Published:     it.Published,
			GUID:          it.GUID,
			Read:          it.IsRead,
			FeedTitleText: it.FeedTitle,
			FeedURL:       it.FeedURL,
		}
	}
	return result
}

// ApplyArticleList updates the article list and title based on feed URL.
func ApplyArticleList(model *list.Model, history *reading.History, feedURL string) {
	model.SetItems(BuildArticleListItems(history, feedURL))
	if feedURL == reading.AllFeedsURL {
		model.Title = "All Feeds"
	} else {
		model.Title = "Articles"
	}
}
