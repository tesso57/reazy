package listview

import (
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FeedItem interface for items that can be rendered by FeedDelegate.
type FeedItem interface {
	list.Item
	Title() string
	URL() string
	IsSectionHeader() bool
}

// FeedDelegate handles rendering of feed items.
type FeedDelegate struct {
	Styles list.DefaultItemStyles
	Theme  lipgloss.Color
}

// NewFeedDelegate creates a new FeedDelegate.
func NewFeedDelegate(themeColor lipgloss.Color) *FeedDelegate {
	return &FeedDelegate{
		Styles: withItemPadding(list.NewDefaultItemStyles()),
		Theme:  themeColor,
	}
}

// Height returns the height of the item.
func (d FeedDelegate) Height() int {
	return 1
}

// Spacing returns the spacing between items.
func (d FeedDelegate) Spacing() int {
	return 0
}

// Update handles messages for the delegate.
func (d FeedDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

// Render renders the item.
func (d FeedDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(FeedItem)
	if !ok {
		return
	}

	title := i.Title()

	style := itemStyle(d.Styles, m, index)
	if i.IsSectionHeader() {
		style = style.Bold(true).Foreground(lipgloss.Color("246"))
	}
	title = truncateItemText(m, style, title)
	renderItemText(w, style, title)
}
