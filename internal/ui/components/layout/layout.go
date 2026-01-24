// Package layout provides the main layout component.
package layout

import (
	"github.com/charmbracelet/lipgloss"
)

// Props defines the properties for the layout component.
type Props struct {
	Sidebar string
	Main    string
	Footer  string
}

// Render renders the layout component.
func Render(p Props) string {
	content := lipgloss.JoinHorizontal(lipgloss.Top, p.Sidebar, p.Main)
	return lipgloss.JoinVertical(lipgloss.Left, content, p.Footer)
}
