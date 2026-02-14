package update

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/tesso57/reazy/internal/presentation/tui/metrics"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
)

type layoutMetrics struct {
	sidebarWidth      int
	mainWidth         int
	sidebarListHeight int
	mainListHeight    int
}

func UpdateListSizes(s *state.ModelState) {
	if s.Width <= 0 || s.Height <= 0 {
		return
	}

	layout := buildLayoutMetrics(s)
	s.FeedList.SetSize(layout.sidebarWidth, layout.sidebarListHeight)
	s.ArticleList.SetSize(layout.mainWidth, layout.mainListHeight)
	s.Viewport.Width = clampMin(layout.mainWidth-1, 1) // main view has left padding of 1
	s.Viewport.Height = layout.mainListHeight
}

func buildLayoutMetrics(s *state.ModelState) layoutMetrics {
	footerHeight := footerHeight(s)
	availableHeight := clampMin(s.Height-footerHeight, 1)

	mainListHeight := clampMin(availableHeight-metrics.HeaderLines, 1)
	sidebarListHeight := clampMin(availableHeight-metrics.SidebarTitleLines, 1)
	if s.Session == state.NewsTopicView {
		mainListHeight = clampMin(mainListHeight-metrics.NewsTopicSummaryLines, 1)
	}

	sidebarWidth := s.Width / 3
	mainWidth := clampMin(s.Width-sidebarWidth-metrics.SidebarRightBorderWidth, 1)

	sidebarListHeight = reservePaginationSpace(s.FeedList, sidebarListHeight)
	mainListHeight = reservePaginationSpace(s.ArticleList, mainListHeight)

	return layoutMetrics{
		sidebarWidth:      sidebarWidth,
		mainWidth:         mainWidth,
		sidebarListHeight: sidebarListHeight,
		mainListHeight:    mainListHeight,
	}
}

func footerHeight(s *state.ModelState) int {
	s.Help.Width = s.Width
	helpText := s.Help.View(&s.Keys)
	return lipgloss.Height(state.FooterText(s.Session, s.Loading, s.AIStatus, s.StatusMessage, helpText))
}

func reservePaginationSpace(m list.Model, height int) int {
	if height < 1 || !m.ShowPagination() {
		return height
	}
	if height <= 1 {
		return height
	}

	statusHeight := 0
	if m.ShowStatusBar() {
		statusHeight = 1
	}

	availHeight := height - statusHeight
	if availHeight < 1 {
		return height
	}

	if len(m.VisibleItems()) > availHeight {
		return height - 1
	}
	return height
}

func clampMin(value, min int) int {
	if value < min {
		return min
	}
	return value
}
