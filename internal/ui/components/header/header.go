package header

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Props struct {
	Visible   bool
	Link      string
	FeedTitle string
}

func Render(p Props) string {
	if !p.Visible {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("🔗 %s\n🏷️  %s\n", p.Link, p.FeedTitle))
}
