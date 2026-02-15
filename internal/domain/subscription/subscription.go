// Package subscription defines feed subscription models.
package subscription

// Subscription represents a single feed subscription.
type Subscription struct {
	URL   string
	Group string
}

// FeedGroup represents a named collection of feed URLs.
type FeedGroup struct {
	Name  string
	Feeds []string
}
