package view

import (
	"github.com/tesso57/reazy/internal/ui/components/header"
	"github.com/tesso57/reazy/internal/ui/components/layout"
	main_view "github.com/tesso57/reazy/internal/ui/components/main"
	"github.com/tesso57/reazy/internal/ui/components/modal"
	"github.com/tesso57/reazy/internal/ui/components/sidebar"
)

type Props struct {
	Sidebar sidebar.Props
	Header  header.Props
	Main    main_view.Props
	Modal   modal.Props
	Footer  string
}

func Render(p Props) string {
	if p.Modal.Visible {
		return modal.Render(p.Modal)
	}

	sidebarStr := sidebar.Render(p.Sidebar)
	headerStr := header.Render(p.Header)

	p.Main.Header = headerStr
	mainStr := main_view.Render(p.Main)

	layoutProps := layout.Props{
		Sidebar: sidebarStr,
		Main:    mainStr,
		Footer:  p.Footer,
	}

	return layout.Render(layoutProps)
}
