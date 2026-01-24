package main_view

import (
	"github.com/charmbracelet/lipgloss"
)

type Props struct {
	Width  int
	Height int
	Header string
	Body   string
}

func Render(p Props) string {
	mainStyle := lipgloss.NewStyle().
		Width(p.Width).
		Height(p.Height).
		PaddingLeft(1)

	return mainStyle.Render(p.Header + p.Body)
}
