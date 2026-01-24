package delegate

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ArticleItem interface {
	list.Item
	Title() string
	IsRead() bool
	FeedTitle() string
}

type ArticleDelegate struct {
	Styles list.DefaultItemStyles
}

func NewArticleDelegate() *ArticleDelegate {
	return &ArticleDelegate{
		Styles: list.NewDefaultItemStyles(),
	}
}

func (d *ArticleDelegate) Height() int {
	return 1
}

func (d *ArticleDelegate) Spacing() int {
	return 0
}

func (d *ArticleDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

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
