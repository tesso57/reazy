// Package tui provides the main user interface model and view components.
package tui

import (
	"fmt"

	"github.com/tesso57/reazy/internal/presentation/tui/components/header"
	main_view "github.com/tesso57/reazy/internal/presentation/tui/components/main"
	"github.com/tesso57/reazy/internal/presentation/tui/components/modal"
	"github.com/tesso57/reazy/internal/presentation/tui/components/sidebar"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
	"github.com/tesso57/reazy/internal/presentation/tui/view"
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
		View:   m.state.FeedList.View(),
		Width:  m.state.FeedList.Width(),
		Height: m.state.FeedList.Height(),
		Active: m.state.Session == state.FeedView,
		Title:  "Reazy Feeds",
	}
}

func (m *Model) buildHeaderProps() header.Props {
	visible := m.state.Session == state.ArticleView || m.state.Session == state.DetailView || m.state.Session == state.FeedView
	var link, feedTitle string

	if visible {
		var currentItem *presenter.Item
		if m.state.Session == state.FeedView {
			if i, ok := m.state.FeedList.SelectedItem().(*presenter.Item); ok {
				currentItem = i
			}
		} else {
			if i, ok := m.state.ArticleList.SelectedItem().(*presenter.Item); ok {
				currentItem = i
			}
		}

		if currentItem != nil {
			// Truncate logic
			availableWidth := m.state.Width - 4
			link = currentItem.Link
			if len(link) > availableWidth && availableWidth > 0 {
				link = link[:availableWidth] + "..."
			}
			// For feed items, title is usually formatted index + title.
			// But header Props expects "FeedTitle".
			// In feedList item, we don't store FeedTitle explicitly?
			// The item struct has feedTitle field.
			// Let's check model.go logic.
			feedTitle = currentItem.FeedTitleText
			// If feedTitle is empty (e.g. initial item for feedList doesn't populate feedTitle?),
			// use Title.
			if feedTitle == "" {
				// In feedList, title is "1. URL". link is URL.
				// We can use link as title if feedTitle is missing.
				// Or m.currentFeed.Title if available and matches?
				// Simple fallback:
				feedTitle = currentItem.TitleText
			}
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
	case m.state.Loading:
		body = fmt.Sprintf("\n\n   %s Loading feed...", m.state.Spinner.View())
	case m.state.Session == state.DetailView:
		body = m.state.Viewport.View()
	case m.state.Session == state.ArticleView || m.state.Session == state.FeedView:
		body = m.state.ArticleList.View()
	default:
		body = ""
	}

	return main_view.Props{
		Width:  m.state.ArticleList.Width(),
		Height: m.state.ArticleList.Height(),
		Header: "", // Will be filled by Render using HeaderProps
		Body:   body,
	}
}

func (m *Model) buildModalProps() modal.Props {
	if m.state.Session == state.AddingFeedView {
		return modal.Props{
			Visible: true,
			Kind:    modal.AddFeed,
			Body: fmt.Sprintf(
				"Enter Feed URL:\n\n%s\n\n(esc to cancel)",
				m.state.TextInput.View(),
			),
			Width:  m.state.Width,
			Height: m.state.Height,
		}
	}
	if m.state.Session == state.QuitView {
		return modal.Props{
			Visible: true,
			Kind:    modal.Quit,
			Body:    "Are you sure you want to quit?\n\n(y/n)",
			Width:   m.state.Width,
			Height:  m.state.Height,
		}
	}
	if m.state.Help.ShowAll {
		return modal.Props{
			Visible: true,
			Kind:    modal.Help,
			Body:    m.state.Help.View(&m.state.Keys),
			Width:   m.state.Width,
			Height:  m.state.Height,
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
	return m.state.Help.View(&m.state.Keys)
}
