package presenter

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/tesso57/reazy/internal/domain/reading"
)

func TestBuildFeedListItems(t *testing.T) {
	items := BuildFeedListItems([]string{
		"https://example.com/feed1.xml",
		"https://example.com/feed2.xml",
	})

	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(items))
	}

	assertItem := func(index int, wantTitle, wantLink string) {
		t.Helper()
		item, ok := items[index].(*Item)
		if !ok {
			t.Fatalf("items[%d] should be *Item", index)
		}
		if item.TitleText != wantTitle {
			t.Fatalf("items[%d].TitleText = %q, want %q", index, item.TitleText, wantTitle)
		}
		if item.Link != wantLink {
			t.Fatalf("items[%d].Link = %q, want %q", index, item.Link, wantLink)
		}
	}

	assertItem(0, "0. * All Feeds", reading.AllFeedsURL)
	assertItem(1, "1. * News", reading.NewsURL)
	assertItem(2, "2. * Bookmarks", reading.BookmarksURL)
	assertItem(3, "3. https://example.com/feed1.xml", "https://example.com/feed1.xml")
	assertItem(4, "4. https://example.com/feed2.xml", "https://example.com/feed2.xml")
}

func TestBuildArticleListItems_AddsDateSections(t *testing.T) {
	tz := time.FixedZone("JST", 9*60*60)
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"guid1": {
			GUID:        "guid1",
			Kind:        reading.ArticleKind,
			Title:       "Article One",
			Link:        "http://example.com/1",
			Date:        time.Date(2026, 2, 14, 10, 0, 0, 0, tz),
			FeedURL:     "http://example.com/feed",
			FeedTitle:   "My Feed",
			Description: "Desc 1",
			Content:     "Content 1",
		},
		"guid2": {
			GUID:        "guid2",
			Kind:        reading.ArticleKind,
			Title:       "Article Two",
			Link:        "http://example.com/2",
			Date:        time.Date(2026, 2, 14, 8, 0, 0, 0, tz),
			FeedURL:     "http://example.com/feed",
			FeedTitle:   "My Feed",
			Description: "Desc 2",
			Content:     "Content 2",
		},
		"guid3": {
			GUID:      "guid3",
			Kind:      reading.ArticleKind,
			Title:     "Older",
			Link:      "http://example.com/3",
			Date:      time.Date(2026, 2, 13, 20, 0, 0, 0, tz),
			FeedURL:   "http://example.com/feed",
			FeedTitle: "My Feed",
		},
		"d1": {
			GUID:       "d1",
			Kind:       reading.NewsDigestKind,
			DigestDate: "2026-02-14",
			FeedURL:    reading.NewsURL,
			Title:      "Digest should be hidden",
		},
	})

	items := BuildArticleListItems(history, "http://example.com/feed")
	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(items))
	}

	section1 := items[0].(*Item)
	if !section1.IsSectionHeader() {
		t.Fatal("first item should be section header")
	}
	if !strings.Contains(section1.TitleText, "2026-02-14") {
		t.Fatalf("section title = %q, want 2026-02-14", section1.TitleText)
	}

	i1 := items[1].(*Item)
	if i1.IsSectionHeader() || !strings.HasPrefix(i1.TitleText, "1. Article One") {
		t.Fatalf("unexpected first article row: %#v", i1)
	}
}

func TestBuildArticleListItems_AllFeedsIncludesFeedName(t *testing.T) {
	now := time.Now()
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"guid1": {
			GUID:      "guid1",
			Kind:      reading.ArticleKind,
			Title:     "Article One",
			FeedURL:   "http://example.com/feed",
			FeedTitle: "My Feed",
			Date:      now,
		},
	})

	items := BuildArticleListItems(history, reading.AllFeedsURL)
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	i := items[1].(*Item)
	if !strings.Contains(i.TitleText, "[My Feed]") {
		t.Fatalf("title should include feed tag in all feeds: %q", i.TitleText)
	}
}

func TestBuildArticleListItems_NewsShowsDigestHistoryByDate(t *testing.T) {
	today := time.Now().In(time.Local).Format("2006-01-02")
	yesterday := time.Now().In(time.Local).Add(-24 * time.Hour).Format("2006-01-02")
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"d_today": {
			GUID:         "d_today",
			Kind:         reading.NewsDigestKind,
			DigestDate:   today,
			Title:        "Today Topic",
			Description:  "Summary",
			RelatedGUIDs: []string{"a1"},
			FeedURL:      reading.NewsURL,
		},
		"d_yesterday": {
			GUID:       "d_yesterday",
			Kind:       reading.NewsDigestKind,
			DigestDate: yesterday,
			Title:      "Yesterday Topic",
			FeedURL:    reading.NewsURL,
		},
		"a1": {
			GUID:    "a1",
			Kind:    reading.ArticleKind,
			Title:   "Article",
			FeedURL: "feed1",
		},
	})

	items := BuildArticleListItems(history, reading.NewsURL)
	if len(items) != 4 {
		t.Fatalf("len(items) = %d, want 4", len(items))
	}
	sectionToday := items[0].(*Item)
	if !sectionToday.IsSectionHeader() || !strings.Contains(sectionToday.TitleText, today) {
		t.Fatalf("first section should be today, got %#v", sectionToday)
	}
	itemToday := items[1].(*Item)
	if !itemToday.IsNewsDigest() {
		t.Fatalf("item should be news digest: %#v", itemToday)
	}
	if itemToday.GUID != "d_today" {
		t.Fatalf("digest guid = %q, want d_today", itemToday.GUID)
	}
	sectionYesterday := items[2].(*Item)
	if !sectionYesterday.IsSectionHeader() || !strings.Contains(sectionYesterday.TitleText, yesterday) {
		t.Fatalf("second section should be yesterday, got %#v", sectionYesterday)
	}
	itemYesterday := items[3].(*Item)
	if itemYesterday.GUID != "d_yesterday" {
		t.Fatalf("digest guid = %q, want d_yesterday", itemYesterday.GUID)
	}
}

func TestBuildArticleListItems_NewsSameDateShowsLatestFirst(t *testing.T) {
	today := time.Now().In(time.Local).Format("2006-01-02")
	now := time.Now().In(time.Local)
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"d_old": {
			GUID:       "d_old",
			Kind:       reading.NewsDigestKind,
			DigestDate: today,
			Title:      "Old Topic",
			Date:       now.Add(-time.Minute),
			FeedURL:    reading.NewsURL,
		},
		"d_new": {
			GUID:       "d_new",
			Kind:       reading.NewsDigestKind,
			DigestDate: today,
			Title:      "New Topic",
			Date:       now,
			FeedURL:    reading.NewsURL,
		},
	})

	items := BuildArticleListItems(history, reading.NewsURL)
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	firstDigest := items[1].(*Item)
	secondDigest := items[2].(*Item)
	if firstDigest.GUID != "d_new" || secondDigest.GUID != "d_old" {
		t.Fatalf("unexpected order: %q then %q", firstDigest.GUID, secondDigest.GUID)
	}
}

func TestApplyArticleList_SelectsFirstNewsDigest(t *testing.T) {
	today := time.Now().In(time.Local).Format("2006-01-02")
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"d1": {
			GUID:       "d1",
			Kind:       reading.NewsDigestKind,
			DigestDate: today,
			Title:      "Topic One",
			FeedURL:    reading.NewsURL,
		},
		"d2": {
			GUID:       "d2",
			Kind:       reading.NewsDigestKind,
			DigestDate: today,
			Title:      "Topic Two",
			FeedURL:    reading.NewsURL,
		},
	})

	model := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	ApplyArticleList(&model, history, reading.NewsURL)

	if model.Title != "News" {
		t.Fatalf("model.Title = %q, want News", model.Title)
	}
	if model.Index() != 1 {
		t.Fatalf("news selected index = %d, want 1", model.Index())
	}
}

func TestApplyRelatedArticleList(t *testing.T) {
	history := reading.NewHistory(map[string]*reading.HistoryItem{
		"a1": {
			GUID:      "a1",
			Kind:      reading.ArticleKind,
			Title:     "Article 1",
			FeedTitle: "Feed 1",
			FeedURL:   "feed1",
		},
		"a2": {
			GUID:      "a2",
			Kind:      reading.ArticleKind,
			Title:     "Article 2",
			FeedTitle: "Feed 2",
			FeedURL:   "feed2",
		},
	})

	model := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 20)
	ApplyRelatedArticleList(&model, history, []string{"a2", "missing", "a1"})

	if model.Title != "Related Articles" {
		t.Fatalf("model.Title = %q, want Related Articles", model.Title)
	}
	if len(model.Items()) != 2 {
		t.Fatalf("items len = %d, want 2", len(model.Items()))
	}
	first := model.Items()[0].(*Item)
	if !strings.Contains(first.TitleText, "[Feed 2]") {
		t.Fatalf("first related item should keep guid order with feed name: %q", first.TitleText)
	}
}
