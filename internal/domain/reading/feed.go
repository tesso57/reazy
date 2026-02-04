// Package reading defines core reading models.
package reading

import "time"

// AllFeedsURL is the special URL used to represent the aggregated "All Feeds" view.
const AllFeedsURL = "internal://all"

// BookmarksURL is the special URL used to represent the filtered "Bookmarks" view.
const BookmarksURL = "internal://bookmarks"

// Item represents a single RSS item.
type Item struct {
	GUID        string
	Title       string
	Link        string
	Published   string
	Description string
	Content     string
	Date        time.Time
	FeedTitle   string
	FeedURL     string
}

// Feed represents a parsed RSS feed.
type Feed struct {
	Title string
	Items []Item
	URL   string
}
