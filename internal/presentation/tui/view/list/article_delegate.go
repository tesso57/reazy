// Package listview provides list item delegates for the view layer.
package listview

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ArticleItem interface for items that can be rendered by ArticleDelegate.
type ArticleItem interface {
	list.Item
	Title() string
	IsRead() bool
	IsBookmarked() bool
	HasAISummary() bool
	FeedTitle() string
}

// ArticleDelegate handles rendering of article items.
type ArticleDelegate struct {
	Styles list.DefaultItemStyles
}

// NewArticleDelegate creates a new ArticleDelegate.
func NewArticleDelegate() *ArticleDelegate {
	return &ArticleDelegate{
		Styles: withItemPadding(list.NewDefaultItemStyles()),
	}
}

// Height returns the height of the item.
func (d *ArticleDelegate) Height() int {
	return 1
}

// Spacing returns the spacing between items.
func (d *ArticleDelegate) Spacing() int {
	return 0
}

// Update handles messages for the delegate.
func (d *ArticleDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

// Render renders the item.
func (d *ArticleDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(ArticleItem)
	if !ok {
		return
	}

	title := decorateArticleTitle(i.Title(), i.IsBookmarked(), i.HasAISummary())

	style := itemStyle(d.Styles, m, index)
	title = truncateItemText(m, style, title)

	// If IsRead, Apply Faint
	if i.IsRead() {
		title = lipgloss.NewStyle().Faint(true).Render(title)
	}

	renderItemText(w, style, title)
}

func decorateArticleTitle(title string, bookmarked, hasAISummary bool) string {
	badges := make([]string, 0, 2)
	if hasAISummary {
		badges = append(badges, "[AI]")
	}
	if bookmarked {
		badges = append(badges, "[B]")
	}
	if len(badges) == 0 {
		return title
	}

	badgeText := strings.Join(badges, " ")
	prefix, rest, ok := splitOrdinalPrefix(title)
	if !ok {
		return fmt.Sprintf("%s %s", badgeText, title)
	}
	if rest == "" {
		return strings.TrimSpace(fmt.Sprintf("%s%s", prefix, badgeText))
	}
	return fmt.Sprintf("%s%s %s", prefix, badgeText, rest)
}

func splitOrdinalPrefix(title string) (prefix, rest string, ok bool) {
	dotIdx := strings.Index(title, ". ")
	if dotIdx <= 0 {
		return "", "", false
	}
	for _, r := range title[:dotIdx] {
		if r < '0' || r > '9' {
			return "", "", false
		}
	}
	return title[:dotIdx+2], strings.TrimLeft(title[dotIdx+2:], " "), true
}
