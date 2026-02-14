// Package tui provides the main user interface model and view components.
package tui

import (
	"fmt"
	"strings"

	"github.com/tesso57/reazy/internal/presentation/tui/components/header"
	main_view "github.com/tesso57/reazy/internal/presentation/tui/components/main"
	"github.com/tesso57/reazy/internal/presentation/tui/components/modal"
	"github.com/tesso57/reazy/internal/presentation/tui/components/sidebar"
	"github.com/tesso57/reazy/internal/presentation/tui/metrics"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
	"github.com/tesso57/reazy/internal/presentation/tui/textutil"
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
	visible := headerVisible(m.state)
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
			sidebarWidth := m.state.Width / 3
			mainWidth := m.state.Width - sidebarWidth - metrics.SidebarRightBorderWidth
			// Main view has 1 padding left. Header has "🔗 " prefix (~3 chars).
			// Safe buffer: metrics.HeaderWidthPadding.
			availableWidth := mainWidth - metrics.HeaderWidthPadding
			link = headerLine(currentItem.Link, availableWidth)
			// For feed items, title is usually formatted index + title.
			// But header Props expects "FeedTitle".
			// In feedList item, we don't store FeedTitle explicitly?
			// The item struct has feedTitle field.
			// Let's check model.go logic.
			feedTitle = headerLine(currentItem.FeedTitleText, availableWidth)
			// If feedTitle is empty (e.g. initial item for feedList doesn't populate feedTitle?),
			// use Title.
			if feedTitle == "" {
				// In feedList, title is "1. URL". link is URL.
				// We can use link as title if feedTitle is missing.
				// Or m.currentFeed.Title if available and matches?
				// Simple fallback:
				feedTitle = headerLine(currentItem.TitleText, availableWidth)
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
		message := "Loading feed..."
		if m.state.AIStatus != "" {
			message = m.state.AIStatus
		}
		body = fmt.Sprintf("\n\n   %s %s", m.state.Spinner.View(), message)
	case m.state.Session == state.DetailView:
		body = m.state.Viewport.View()
	case m.state.Session == state.NewsTopicView:
		body = buildNewsTopicBody(m.state)
	case m.state.Session == state.ArticleView || m.state.Session == state.FeedView:
		body = m.state.ArticleList.View()
	default:
		body = ""
	}
	if m.state.Err != nil && (m.state.Session == state.ArticleView || m.state.Session == state.FeedView) && !m.state.Loading {
		body = fmt.Sprintf("Error: %v\n\n%s", m.state.Err, body)
	}

	headerHeight := 0
	if headerVisible(m.state) {
		headerHeight = metrics.HeaderLines
	}

	return main_view.Props{
		Width:  m.state.ArticleList.Width(),
		Height: m.state.ArticleList.Height() + headerHeight,
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
	if m.state.Session == state.DeleteFeedView {
		return modal.Props{
			Visible: true,
			Kind:    modal.DeleteFeed,
			Body:    "Are you sure you want to delete this feed?\n\n(y/n)",
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
	helpText := m.state.Help.View(&m.state.Keys)
	return state.FooterText(m.state.Session, m.state.Loading, m.state.AIStatus, m.state.StatusMessage, helpText)
}

func headerVisible(st *state.ModelState) bool {
	if st == nil {
		return false
	}
	switch st.Session {
	case state.FeedView, state.DetailView:
		return true
	case state.ArticleView:
		if item, ok := st.ArticleList.SelectedItem().(*presenter.Item); ok && item != nil && item.IsSectionHeader() {
			return false
		}
		if item, ok := st.ArticleList.SelectedItem().(*presenter.Item); ok && item != nil && item.IsNewsDigest() {
			return false
		}
		return true
	case state.NewsTopicView:
		return false
	default:
		return false
	}
}

func headerLine(text string, width int) string {
	return textutil.Truncate(textutil.SingleLine(text), width)
}

func buildNewsTopicBody(st *state.ModelState) string {
	if st == nil {
		return ""
	}
	title := strings.TrimSpace(st.NewsTopicTitle)
	if title == "" {
		title = "Daily News Topic"
	}
	summary := strings.TrimSpace(st.NewsTopicSummary)
	if summary == "" {
		summary = "(No summary available.)"
	}
	tags := ""
	if len(st.NewsTopicTags) > 0 {
		tags = fmt.Sprintf("Tags: %s\n", strings.Join(st.NewsTopicTags, ", "))
	}

	return fmt.Sprintf(
		"%s\n----------------------------------------\n%s%s\nRelated Articles\n%s",
		title,
		tags,
		summary,
		st.ArticleList.View(),
	)
}
