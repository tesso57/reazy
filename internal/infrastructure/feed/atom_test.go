package feed

import (
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
)

func TestAtomParsing(t *testing.T) {
	// Sample Atom feed content from https://github.com/golang/go/releases.atom
	atomContent := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/" xml:lang="en-US">
  <id>tag:github.com,2008:https://github.com/golang/go/releases</id>
  <link type="text/html" rel="alternate" href="https://github.com/golang/go/releases"/>
  <link type="application/atom+xml" rel="self" href="https://github.com/golang/go/releases.atom"/>
  <title>Release notes from go</title>
  <updated>2026-01-15T18:32:04Z</updated>
  <entry>
    <id>tag:github.com,2008:Repository/23096959/go1.26rc2</id>
    <updated>2026-01-15T18:32:04Z</updated>
    <link rel="alternate" type="text/html" href="https://github.com/golang/go/releases/tag/go1.26rc2"/>
    <title>[release-branch.go1.26] go1.26rc2</title>
    <content type="html">&lt;p&gt;Change-Id: If5ce85a68010848f16c4c2509e18466ed1356912&lt;/p&gt;</content>
    <author>
      <name>gopherbot</name>
    </author>
    <media:thumbnail height="30" width="30" url="https://avatars.githubusercontent.com/u/8566911?s=60&amp;v=4"/>
  </entry>
</feed>`

	// Mock the ParserFunc to return the parsed atom content
	originalParser := ParserFunc
	defer func() { ParserFunc = originalParser }()

	ParserFunc = func(url string) (*gofeed.Feed, error) {
		fp := gofeed.NewParser()
		return fp.ParseString(atomContent)
	}

	feed, err := Fetch("https://github.com/golang/go/releases.atom")
	if err != nil {
		t.Fatalf("Failed to fetch atom feed: %v", err)
	}

	if feed.Title != "Release notes from go" {
		t.Errorf("Expected title 'Release notes from go', got '%s'", feed.Title)
	}

	if len(feed.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(feed.Items))
	}

	item := feed.Items[0]
	if item.Title != "[release-branch.go1.26] go1.26rc2" {
		t.Errorf("Expected item title '[release-branch.go1.26] go1.26rc2', got '%s'", item.Title)
	}

	expectedDate, _ := time.Parse(time.RFC3339, "2026-01-15T18:32:04Z")
	if !item.Date.Equal(expectedDate) {
		t.Errorf("Expected date %v, got %v", expectedDate, item.Date)
	}
}
