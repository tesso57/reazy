package sidebar

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		props    Props
		wantView string
	}{
		{
			name: "Active",
			props: Props{
				View:   "FEED LIST",
				Width:  20,
				Height: 10,
				Active: true,
			},
			wantView: "FEED LIST",
		},
		{
			name: "Inactive",
			props: Props{
				View:   "FEED LIST",
				Width:  20,
				Height: 10,
				Active: false,
			},
			wantView: "FEED LIST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.props)
			if !strings.Contains(got, tt.wantView) {
				t.Errorf("Render() = %q, want content %q", got, tt.wantView)
			}
		})
	}
}
