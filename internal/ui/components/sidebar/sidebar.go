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

	return sidebarStyle.Render(p.View)
}
