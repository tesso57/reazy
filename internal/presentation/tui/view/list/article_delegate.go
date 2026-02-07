// Package listview provides list item delegates for the view layer.
package listview

import (
	"fmt"
	"io"

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

	title := i.Title()

	// If Bookmarked, prepend [B]
	if i.IsBookmarked() {
		title = fmt.Sprintf("[B] %s", title)
	}
	if i.HasAISummary() {
		title = fmt.Sprintf("[AI] %s", title)
	}

	style := itemStyle(d.Styles, m, index)
	title = truncateItemText(m, style, title)

	// If IsRead, Apply Faint
	if i.IsRead() {
		title = lipgloss.NewStyle().Faint(true).Render(title)
	}

	renderItemText(w, style, title)
}
