package view

import (
	"strings"
	"testing"

	"github.com/tesso57/reazy/internal/ui/components/header"
	main_view "github.com/tesso57/reazy/internal/ui/components/main"
	"github.com/tesso57/reazy/internal/ui/components/modal"
	"github.com/tesso57/reazy/internal/ui/components/sidebar"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name      string
		props     Props
		wantExact string   // if set, check exact string (for modal)
		wantParts []string // check containment
	}{
		{
			name: "Modal Overlay",
			props: Props{
				Modal: modal.Props{
					Visible: true,
					Kind:    modal.Help,
					Body:    "HELP_CONTENT",
					Width:   100,
					Height:  50,
				},
			},
			wantParts: []string{"HELP_CONTENT"},
		},
		{
			name: "Standard Layout",
			props: Props{
				Sidebar: sidebar.Props{
					View:   "SIDEBAR_CONTENT",
					Width:  20,
					Height: 10,
				},
				Header: header.Props{
					Visible:   true,
					Link:      "LINK",
					FeedTitle: "FEED",
				},
				Main: main_view.Props{
					Width:  80,
					Height: 10,
					Body:   "MAIN_CONTENT",
				},
				Footer: "FOOTER_HELP",
			},
			wantParts: []string{
				"SIDEBAR_CONTENT",
				"LINK",
				"FEED",
				"MAIN_CONTENT",
				"FOOTER_HELP",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.props)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("Render() expected to contain %q", part)
				}
			}
		})
	}
}
