package state

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/tesso57/reazy/internal/application/settings"
)

func TestFooterText(t *testing.T) {
	tests := []struct {
		name          string
		session       Session
		loading       bool
		aiStatus      string
		statusMessage string
		helpText      string
		want          string
	}{
		{
			name:     "help only when no ai status",
			session:  ArticleView,
			helpText: "help",
			want:     "help",
		},
		{
			name:     "help only outside article/detail",
			session:  FeedView,
			aiStatus: "AI: updated",
			helpText: "help",
			want:     "help",
		},
		{
			name:          "generic status shown in feed view",
			session:       FeedView,
			statusMessage: "2 feeds timed out",
			helpText:      "help",
			want:          "2 feeds timed out\nhelp",
		},
		{
			name:     "help only while loading",
			session:  ArticleView,
			loading:  true,
			aiStatus: "AI: updated",
			helpText: "help",
			want:     "help",
		},
		{
			name:     "ai status prepended in article view",
			session:  ArticleView,
			aiStatus: "AI: updated",
			helpText: "help",
			want:     "AI: updated\nhelp",
		},
		{
			name:     "ai status prepended in detail view",
			session:  DetailView,
			aiStatus: "AI: updated",
			helpText: "help",
			want:     "AI: updated\nhelp",
		},
		{
			name:     "ai status only when help empty",
			session:  DetailView,
			aiStatus: "AI: updated",
			helpText: "",
			want:     "AI: updated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FooterText(tt.session, tt.loading, tt.aiStatus, tt.statusMessage, tt.helpText)
			if got != tt.want {
				t.Fatalf("FooterText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFooterHelpText_TwoLines(t *testing.T) {
	keys := NewKeyMap(settings.KeyMapConfig{
		Up:         "k",
		Down:       "j",
		Left:       "h",
		Right:      "l",
		Open:       "enter",
		Back:       "esc",
		Quit:       "q",
		GroupFeeds: "z",
	})

	got := FooterHelpText(help.New(), keys)
	if strings.Count(got, "\n") != 1 {
		t.Fatalf("FooterHelpText() should be two lines, got %q", got)
	}
}
