package modal

import (
	"github.com/charmbracelet/lipgloss"
)

type Kind int

const (
	None Kind = iota
	AddFeed
	Help
)

type Props struct {
	Visible bool
	Kind    Kind
	Body    string
	Width   int
	Height  int
}

func Render(p Props) string {
	if !p.Visible {
		return ""
	}

	borderColor := lipgloss.Color("63") // Default (Help)
	var content string

	if p.Kind == AddFeed {
		borderColor = lipgloss.Color("205")
		// For AddFeed, Body usually contains the full dialog content constructed in container
		// containing title, input view, etc.
		// But in original code:
		// Render(fmt.Sprintf("Enter Feed URL:\n\n%s\n\n(esc to cancel)", m.textInput.View()))
		// So Body here is that whole string.
		content = lipgloss.NewStyle().
			Width(40).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2).
			Render(p.Body)
	} else {
		// Help
		content = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2).
			Render(p.Body)
	}

	return lipgloss.Place(p.Width, p.Height, lipgloss.Center, lipgloss.Center, content)
}
