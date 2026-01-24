package delegate

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FeedItem interface {
	list.Item
	Title() string
	URL() string
}

type FeedDelegate struct {
	Styles list.DefaultItemStyles
	Theme  lipgloss.Color
}

func NewFeedDelegate(themeColor lipgloss.Color) *FeedDelegate {
	return &FeedDelegate{
		Styles: list.NewDefaultItemStyles(),
		Theme:  themeColor,
	}
}

func (d FeedDelegate) Height() int {
	return 1
}

func (d FeedDelegate) Spacing() int {
	return 0
}

func (d FeedDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d FeedDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(FeedItem)
	if !ok {
		return
	}

	title := i.Title()

	// Apply styles based on selection
	if index == m.Index() {
		title = d.Styles.SelectedTitle.Render(title)
	} else {
		title = d.Styles.NormalTitle.Render(title)
	}

	_, _ = fmt.Fprint(w, title)
}
