// Package ui provides the main user interface model and view components.
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/tesso57/reazy/internal/ui/components/header"
	main_view "github.com/tesso57/reazy/internal/ui/components/main"
	"github.com/tesso57/reazy/internal/ui/components/modal"
	"github.com/tesso57/reazy/internal/ui/components/sidebar"
	"github.com/tesso57/reazy/internal/ui/view"
)

func (m *Model) buildProps() view.Props {
	return view.Props{
		Sidebar: m.buildSidebarProps(),
		Header:  m.buildHeaderProps(),
		Main:    m.buildMainProps(),
		Modal:   m.buildModalProps(),
		Footer:  m.buildFooterProps(),
	}
}

func (m *Model) buildSidebarProps() sidebar.Props {
	return sidebar.Props{
		View:   m.feedList.View(),
		Width:  m.feedList.Width(),
		Height: m.feedList.Height(),
		Active: m.state == feedView,
	}
}

func (m *Model) buildHeaderProps() header.Props {
	visible := m.state == articleView || m.state == detailView
	var link, feedTitle string

	if visible {
		if i, ok := m.articleList.SelectedItem().(*item); ok {
			// Truncate logic
			availableWidth := m.width - 4
			link = i.link
			if len(link) > availableWidth && availableWidth > 0 {
				link = link[:availableWidth] + "..."
			}
			feedTitle = i.feedTitle
		}
	}

	return header.Props{
		Visible:   visible,
		Link:      link,
		FeedTitle: feedTitle,
	}
}

func (m *Model) buildMainProps() main_view.Props {
	var body string
	switch {
	case m.loading:
		body = fmt.Sprintf("\n\n   %s Loading feed...", m.spinner.View())
	case m.state == detailView:
		body = m.viewport.View()
	case m.state == articleView:
		body = m.articleList.View()
	default:
		body = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("\n\n  ← Select a feed from the left list.")
	}

	return main_view.Props{
		Width:  m.articleList.Width(),
		Height: m.articleList.Height(),
		Header: "", // Will be filled by Render using HeaderProps
		Body:   body,
	}
}

func (m *Model) buildModalProps() modal.Props {
	if m.state == addingFeedView {
		return modal.Props{
			Visible: true,
			Kind:    modal.AddFeed,
			Body: fmt.Sprintf(
				"Enter Feed URL:\n\n%s\n\n(esc to cancel)",
				m.textInput.View(),
			),
			Width:  m.width,
			Height: m.height,
		}
	}
	if m.help.ShowAll {
		return modal.Props{
			Visible: true,
			Kind:    modal.Help,
			Body:    m.help.View(&m.keys),
			Width:   m.width,
			Height:  m.height,
		}
	}
	return modal.Props{Visible: false}
}

func (m *Model) buildFooterProps() string {
	// If modal is showing help, we might not want to show footer help?
	// But in model.go logic:
	// if m.help.ShowAll { return modal } -> exits early, no footer rendering loop.
	// So footer string isn't used if modal is active.
	// buildProps generates it anyway.
	return m.help.View(&m.keys)
}
