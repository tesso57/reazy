// Package ui provides the main user interface model and view components.
package ui

import (
	"fmt"

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
		Title:  "Reazy Feeds",
	}
}

func (m *Model) buildHeaderProps() header.Props {
	visible := m.state == articleView || m.state == detailView || m.state == feedView
	var link, feedTitle string

	if visible {
		var currentItem *item
		if m.state == feedView {
			if i, ok := m.feedList.SelectedItem().(*item); ok {
				currentItem = i
			}
		} else {
			if i, ok := m.articleList.SelectedItem().(*item); ok {
				currentItem = i
			}
		}

		if currentItem != nil {
			// Truncate logic
			availableWidth := m.width - 4
			link = currentItem.link
			if len(link) > availableWidth && availableWidth > 0 {
				link = link[:availableWidth] + "..."
			}
			// For feed items, title is usually formatted index + title.
			// But header Props expects "FeedTitle".
			// In feedList item, we don't store FeedTitle explicitly?
			// The item struct has feedTitle field.
			// Let's check model.go logic.
			feedTitle = currentItem.feedTitle
			// If feedTitle is empty (e.g. initial item for feedList doesn't populate feedTitle?),
			// use Title.
			if feedTitle == "" {
				// In feedList, title is "1. URL". link is URL.
				// We can use link as title if feedTitle is missing.
				// Or m.currentFeed.Title if available and matches?
				// Simple fallback:
				feedTitle = currentItem.title
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
	case m.loading:
		body = fmt.Sprintf("\n\n   %s Loading feed...", m.spinner.View())
	case m.state == detailView:
		body = m.viewport.View()
	case m.state == articleView || m.state == feedView:
		body = m.articleList.View()
	default:
		body = ""
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
	if m.state == quitView {
		return modal.Props{
			Visible: true,
			Kind:    modal.Quit,
			Body:    "Are you sure you want to quit?\n\n(y/n)",
			Width:   m.width,
			Height:  m.height,
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
