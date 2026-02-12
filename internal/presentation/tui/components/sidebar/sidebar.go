// Package sidebar provides the sidebar component.
package sidebar

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tesso57/reazy/internal/presentation/tui/textutil"
)

// Props defines the properties for the sidebar component.
type Props struct {
	View   string
	Width  int
	Height int
	Title  string
	Active bool
}

// Render renders the sidebar component.
func Render(p Props) string {
	sidebarStyle := lipgloss.NewStyle().
		Width(p.Width).
		Height(p.Height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("63"))

	if p.Active {
		sidebarStyle = sidebarStyle.BorderForeground(lipgloss.Color("205"))
	}

	titleStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingBottom(1).
		Foreground(lipgloss.Color("205")) // Use same color as spinner/active border for consistency

	titleWidth := p.Width - 3 // reserve right border + left padding
	if titleWidth < 0 {
		titleWidth = 0
	}
	title := textutil.Truncate(textutil.SingleLine(p.Title), titleWidth)

	return sidebarStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		p.View,
	))
}
