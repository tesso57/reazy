package reading

import "testing"

func TestIsVirtualFeedURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "all feeds", url: AllFeedsURL, want: true},
		{name: "news", url: NewsURL, want: true},
		{name: "bookmarks", url: BookmarksURL, want: true},
		{name: "custom", url: "https://example.com/rss", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVirtualFeedURL(tt.url); got != tt.want {
				t.Fatalf("IsVirtualFeedURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestFeedKinds(t *testing.T) {
	if ArticleKind == "" {
		t.Fatal("ArticleKind should not be empty")
	}
	if NewsDigestKind == "" {
		t.Fatal("NewsDigestKind should not be empty")
	}
	if ArticleKind == NewsDigestKind {
		t.Fatal("kinds must be distinct")
	}
}
