package layout

import (
	"github.com/charmbracelet/lipgloss"
)

type Props struct {
	Sidebar string
	Main    string
	Footer  string
}

func Render(p Props) string {
	content := lipgloss.JoinHorizontal(lipgloss.Top, p.Sidebar, p.Main)
	return lipgloss.JoinVertical(lipgloss.Left, content, p.Footer)
}
