package modal

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name      string
		props     Props
		wantBody  string
		wantVis   bool
		checkType Kind
	}{
		{
			name: "Hidden",
			props: Props{
				Visible: false,
			},
			wantVis: false,
		},
		{
			name: "Help Modal",
			props: Props{
				Visible: true,
				Kind:    Help,
				Body:    "HELP INFO",
				Width:   100,
				Height:  50,
			},
			wantBody: "HELP INFO",
			wantVis:  true,
		},
		{
			name: "Add Feed Modal",
			props: Props{
				Visible: true,
				Kind:    AddFeed,
				Body:    "INPUT URL",
				Width:   100,
				Height:  50,
			},
			wantBody: "INPUT URL",
			wantVis:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.props)
			if !tt.wantVis {
				if got != "" {
					t.Errorf("Render() = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantBody) {
				t.Errorf("Render() = %q, want body %q", got, tt.wantBody)
			}
			// We can check border colors loosely if we want, but purely ensuring content is there is good start.
			// Lipgloss output contains ANSI codes which are brittle to assert exactly.
		})
	}
}
