// Package header provides the module header component.
package header

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Props defines the properties for the header component.
type Props struct {
	Visible   bool
	Link      string
	FeedTitle string
}

// Render renders the header component.
func Render(p Props) string {
	if !p.Visible {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("🔗 %s\n🏷️  %s", p.Link, p.FeedTitle))
}
