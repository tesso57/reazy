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
	s.Viewport.Width = s.Width
	s.Viewport.Height = layout.mainListHeight
}

func buildLayoutMetrics(s *state.ModelState) layoutMetrics {
	footerHeight := footerHeight(s)
	availableHeight := clampMin(s.Height-footerHeight, 1)

	mainListHeight := clampMin(availableHeight-metrics.HeaderLines, 1)
	sidebarListHeight := clampMin(availableHeight-metrics.SidebarTitleLines, 1)

	sidebarWidth := s.Width / 3
	mainWidth := s.Width - sidebarWidth

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
	return lipgloss.Height(s.Help.View(&s.Keys))
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
