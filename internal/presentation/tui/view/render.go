// Package view orchestrates the composition of UI components.
package view

import (
	"github.com/tesso57/reazy/internal/presentation/tui/components/header"
	"github.com/tesso57/reazy/internal/presentation/tui/components/layout"
	mainview "github.com/tesso57/reazy/internal/presentation/tui/components/main"
	"github.com/tesso57/reazy/internal/presentation/tui/components/modal"
	"github.com/tesso57/reazy/internal/presentation/tui/components/sidebar"
)

// Props aggregates properties for all UI components.
type Props struct {
	Sidebar sidebar.Props
	Header  header.Props
	Main    mainview.Props
	Modal   modal.Props
	Footer  string
}

// Render renders the complete UI view based on the provided props.
func Render(p Props) string {
	if p.Modal.Visible {
		return modal.Render(p.Modal)
	}

	sidebarStr := sidebar.Render(p.Sidebar)
	headerStr := header.Render(p.Header)

	p.Main.Header = headerStr
	mainStr := mainview.Render(p.Main)

	layoutProps := layout.Props{
		Sidebar: sidebarStr,
		Main:    mainStr,
		Footer:  p.Footer,
	}

	return layout.Render(layoutProps)
}
