// Package reading defines core reading models.
package reading

import "time"

// AllFeedsURL is the special URL used to represent the aggregated "All Feeds" view.
const AllFeedsURL = "internal://all"

// NewsURL is the special URL used to represent the aggregated "News" view.
const NewsURL = "internal://news"

// BookmarksURL is the special URL used to represent the filtered "Bookmarks" view.
const BookmarksURL = "internal://bookmarks"

const (
	// ArticleKind is the default history item kind.
	ArticleKind = "article"
	// NewsDigestKind is the history item kind for generated daily news digests.
	NewsDigestKind = "news_digest"
)

// IsVirtualFeedURL returns true when the URL is one of the built-in feed tabs.
func IsVirtualFeedURL(url string) bool {
	switch url {
	case AllFeedsURL, NewsURL, BookmarksURL:
		return true
	default:
		return false
	}
}

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
