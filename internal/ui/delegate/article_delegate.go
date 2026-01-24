// Package delegate provides custom list item delegates.
package delegate

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
	FeedTitle() string
}

// ArticleDelegate handles rendering of article items.
type ArticleDelegate struct {
	Styles list.DefaultItemStyles
}

// NewArticleDelegate creates a new ArticleDelegate.
func NewArticleDelegate() *ArticleDelegate {
	return &ArticleDelegate{
		Styles: list.NewDefaultItemStyles(),
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

	// Prepend FeedTitle if available (logic migrated from model.go)
	// Actually, the Title() method on the item itself already contained the formatted title
	// in the previous implementation. We should move that logic here if possible,
	// but to keep it simple, let's assume item.Title() returns the main text.
	// In the original code, the Item struct had a customized Title() but logic for prepending
	// [FeedName] was in handleFeedFetchedMsg and baked into the item.title string.
	// Ideally we move that dynamic composition here.

	// If IsRead, Apply Faint
	if i.IsRead() {
		title = lipgloss.NewStyle().Faint(true).Render(title)
	}

	// Apply Selection Styles
	if index == m.Index() {
		title = d.Styles.SelectedTitle.Render(title)
	} else {
		title = d.Styles.NormalTitle.Render(title)
	}

	_, _ = fmt.Fprint(w, title)
}
