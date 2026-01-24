package header

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		props    Props
		wantLink string
		wantFeed string
		wantVis  bool
	}{
		{
			name: "Visible",
			props: Props{
				Visible:   true,
				Link:      "http://example.com",
				FeedTitle: "Example Feed",
			},
			wantLink: "http://example.com",
			wantFeed: "Example Feed",
			wantVis:  true,
		},
		{
			name: "Hidden",
			props: Props{
				Visible: false,
			},
			wantVis: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.props)
			if !tt.wantVis {
				if got != "" {
					t.Errorf("Render() = %q, want empty string", got)
				}
				return
			}

			if !strings.Contains(got, tt.wantLink) {
				t.Errorf("Render() = %q, want link %q", got, tt.wantLink)
			}
			if !strings.Contains(got, tt.wantFeed) {
				t.Errorf("Render() = %q, want feed title %q", got, tt.wantFeed) // Fixed typo in error message
			}
			if !strings.Contains(got, "🔗") {
				t.Error("Render() missing link icon")
			}
		})
	}
}
