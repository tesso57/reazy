// Package sidebar provides the sidebar component.
package sidebar

import (
	"github.com/charmbracelet/lipgloss"
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

	return sidebarStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(p.Title),
		p.View,
	))
}
