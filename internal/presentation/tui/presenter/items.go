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
	Kind          string
	RelatedGUIDs  []string
	SectionHeader bool
	BodyHydrated  bool
}

const (
	// BuiltinAllFeedsListIndex is the sidebar index of the built-in "All Feeds" tab.
	BuiltinAllFeedsListIndex = iota
	// BuiltinNewsListIndex is the sidebar index of the built-in "News" tab.
	BuiltinNewsListIndex
	// BuiltinBookmarksListIndex is the sidebar index of the built-in "Bookmarks" tab.
	BuiltinBookmarksListIndex
	// BuiltinFeedItemCount is the number of non-removable built-in feed tabs.
	BuiltinFeedItemCount
)

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

// IsSectionHeader returns true when the item is a non-article section marker.
func (i *Item) IsSectionHeader() bool { return i.SectionHeader }

// IsNewsDigest returns true when the item is a generated news digest topic.
func (i *Item) IsNewsDigest() bool { return i != nil && i.Kind == reading.NewsDigestKind }

// Description returns a formatted description for list display.
func (i *Item) Description() string {
	if i.Published != "" {
		return fmt.Sprintf("%s - %s", i.Published, i.Desc)
	}
	return i.Desc
}

// BuildFeedListItems builds list items for the feed list.
func BuildFeedListItems(feeds []string) []list.Item {
	items := make([]list.Item, len(feeds)+BuiltinFeedItemCount)
	items[BuiltinAllFeedsListIndex] = &Item{
		TitleText: "0. * All Feeds",
		RawTitle:  "All Feeds",
		Link:      reading.AllFeedsURL,
	}
	items[BuiltinNewsListIndex] = &Item{
		TitleText: "1. * News",
		RawTitle:  "News",
		Link:      reading.NewsURL,
	}
	items[BuiltinBookmarksListIndex] = &Item{
		TitleText: "2. * Bookmarks",
		RawTitle:  "Bookmarks",
		Link:      reading.BookmarksURL,
	}

	for i, f := range feeds {
		index := i + BuiltinFeedItemCount
		items[index] = &Item{
			TitleText: fmt.Sprintf("%d. %s", index, textutil.SingleLine(f)),
			RawTitle:  f,
			Link:      f,
		}
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

	if feedURL == reading.NewsURL {
		return buildNewsDigestListItems(history.DigestItems())
	}

	items := history.ItemsByFeed(feedURL)
	sort.Slice(items, func(i, j int) bool {
		return articleSortDate(items[i]).After(articleSortDate(items[j]))
	})

	return buildDateSectionedArticleListItems(items, feedURL == reading.AllFeedsURL || feedURL == reading.BookmarksURL)
}

// ApplyArticleList updates the article list and title based on feed URL.
func ApplyArticleList(model *list.Model, history *reading.History, feedURL string) {
	model.SetItems(BuildArticleListItems(history, feedURL))
	if feedURL == reading.AllFeedsURL {
		model.Title = "All Feeds"
	} else if feedURL == reading.NewsURL {
		model.Title = "News"
		selectFirstSelectableItem(model)
	} else if feedURL == reading.BookmarksURL {
		model.Title = "Bookmarks"
	} else {
		model.Title = "Articles"
	}
}

// ApplyRelatedArticleList updates the list with related article items.
func ApplyRelatedArticleList(model *list.Model, history *reading.History, relatedGUIDs []string) {
	if model == nil || history == nil {
		return
	}
	related := history.RelatedItems(&reading.HistoryItem{
		Kind:         reading.NewsDigestKind,
		RelatedGUIDs: relatedGUIDs,
	})
	result := make([]list.Item, 0, len(related))
	for index, it := range related {
		result = append(result, buildArticleItem(index+1, it, true))
	}
	model.SetItems(result)
	model.Title = "Related Articles"
	selectFirstSelectableItem(model)
}

func buildArticleItem(index int, it *reading.HistoryItem, showFeedTitle bool) *Item {
	title := textutil.SingleLine(it.Title)
	feedTitle := textutil.SingleLine(it.FeedTitle)
	if showFeedTitle && feedTitle != "" {
		title = fmt.Sprintf("%d. [%s] %s", index, feedTitle, title)
	} else {
		title = fmt.Sprintf("%d. %s", index, title)
	}

	return &Item{
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
		Kind:          kindOrDefault(it.Kind),
		RelatedGUIDs:  append([]string(nil), it.RelatedGUIDs...),
		BodyHydrated:  it.BodyHydrated,
	}
}

func buildNewsDigestListItems(items []*reading.HistoryItem) []list.Item {
	if len(items) == 0 {
		return nil
	}

	sorted := append([]*reading.HistoryItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		leftKey, _ := newsDigestDateKeyAndLabel(sorted[i])
		rightKey, _ := newsDigestDateKeyAndLabel(sorted[j])
		if leftKey == rightKey {
			return sorted[i].GUID < sorted[j].GUID
		}
		if leftKey == unknownDateKey {
			return false
		}
		if rightKey == unknownDateKey {
			return true
		}
		return leftKey > rightKey
	})

	groups := groupItemsByDate(sorted, newsDigestDateKeyAndLabel)
	return buildSectionedListItems(groups, len(sorted), buildNewsDigestItem)
}

type articleGroup struct {
	label string
	items []*reading.HistoryItem
}

const (
	unknownDateKey   = "unknown"
	unknownDateLabel = "Unknown Date"
)

func buildDateSectionedArticleListItems(items []*reading.HistoryItem, showFeedTitle bool) []list.Item {
	if len(items) == 0 {
		return nil
	}
	groups := groupItemsByDate(items, articleDateKeyAndLabel)
	return buildSectionedListItems(groups, len(items), func(index int, it *reading.HistoryItem) *Item {
		return buildArticleItem(index, it, showFeedTitle)
	})
}

func articleSortDate(item *reading.HistoryItem) time.Time {
	if item == nil {
		return time.Time{}
	}
	if !item.Date.IsZero() {
		return item.Date.In(time.Local)
	}
	if !item.SavedAt.IsZero() {
		return item.SavedAt.In(time.Local)
	}
	return time.Time{}
}

func articleDateKeyAndLabel(item *reading.HistoryItem) (string, string) {
	date := articleSortDate(item)
	if date.IsZero() {
		return unknownDateKey, unknownDateLabel
	}
	return date.Format("2006-01-02"), date.Format("2006-01-02 (Mon)")
}

func newsDigestDateKeyAndLabel(item *reading.HistoryItem) (string, string) {
	if item == nil {
		return unknownDateKey, unknownDateLabel
	}
	key := strings.TrimSpace(item.DigestDate)
	if key == "" && !item.SavedAt.IsZero() {
		key = item.SavedAt.In(time.Local).Format("2006-01-02")
	}
	if key == "" && !item.Date.IsZero() {
		key = item.Date.In(time.Local).Format("2006-01-02")
	}
	if key == "" {
		return unknownDateKey, unknownDateLabel
	}

	date, err := time.ParseInLocation("2006-01-02", key, time.Local)
	if err != nil {
		return key, key
	}
	return key, date.Format("2006-01-02 (Mon)")
}

func buildNewsDigestItem(index int, it *reading.HistoryItem) *Item {
	title := textutil.SingleLine(it.Title)
	if title == "" {
		title = "Untitled Topic"
	}
	return &Item{
		TitleText:     fmt.Sprintf("%d. %s", index, title),
		RawTitle:      it.Title,
		Desc:          strings.TrimSpace(it.Description),
		Content:       strings.TrimSpace(it.Content),
		Published:     it.Published,
		GUID:          it.GUID,
		AITags:        append([]string(nil), it.AITags...),
		AIUpdatedAt:   it.AIUpdatedAt,
		FeedTitleText: "Daily News",
		FeedURL:       reading.NewsURL,
		Kind:          reading.NewsDigestKind,
		RelatedGUIDs:  append([]string(nil), it.RelatedGUIDs...),
		BodyHydrated:  true,
	}
}

func buildSectionHeaderItem(label string, count int) *Item {
	return &Item{
		TitleText:     fmt.Sprintf("== %s (%d) ==", label, count),
		RawTitle:      label,
		FeedTitleText: label,
		SectionHeader: true,
	}
}

func buildSectionedListItems(
	groups []articleGroup,
	itemCount int,
	rowBuilder func(index int, it *reading.HistoryItem) *Item,
) []list.Item {
	result := make([]list.Item, 0, itemCount+len(groups))
	index := 1
	for _, group := range groups {
		result = append(result, buildSectionHeaderItem(group.label, len(group.items)))
		for _, it := range group.items {
			result = append(result, rowBuilder(index, it))
			index++
		}
	}
	return result
}

func groupItemsByDate(
	items []*reading.HistoryItem,
	keyLabel func(item *reading.HistoryItem) (string, string),
) []articleGroup {
	result := make([]articleGroup, 0)
	currentKey := ""

	for _, it := range items {
		key, label := keyLabel(it)
		if len(result) == 0 || key != currentKey {
			currentKey = key
			result = append(result, articleGroup{label: label})
		}
		last := len(result) - 1
		result[last].items = append(result[last].items, it)
	}

	return result
}

func selectFirstSelectableItem(model *list.Model) {
	items := model.Items()
	for index, listItem := range items {
		item, ok := listItem.(*Item)
		if ok && !item.IsSectionHeader() {
			model.Select(index)
			return
		}
	}
}

func kindOrDefault(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return reading.ArticleKind
	}
	return kind
}
