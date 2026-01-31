// Package mainview provides the main content area component.
package mainview

import (
	"github.com/charmbracelet/lipgloss"
)

// Props defines the properties for the main view component.
type Props struct {
	Width  int
	Height int
	Header string
	Body   string
}

// Render renders the main view component.
func Render(p Props) string {
	mainStyle := lipgloss.NewStyle().
		Width(p.Width).
		Height(p.Height).
		PaddingLeft(1)

	return mainStyle.Render(p.Header + p.Body)
}
