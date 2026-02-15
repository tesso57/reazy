package state

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// FooterText returns the footer content for the current session.
func FooterText(session Session, loading bool, aiStatus, statusMessage, helpText string) string {
	lines := make([]string, 0, 3)
	if !loading {
		if msg := strings.TrimSpace(statusMessage); msg != "" {
			lines = append(lines, msg)
		}
		if ai := strings.TrimSpace(aiStatus); ai != "" && (session == ArticleView || session == NewsTopicView || session == DetailView) {
			lines = append(lines, ai)
		}
	}
	if h := strings.TrimSpace(helpText); h != "" {
		lines = append(lines, h)
	}
	return strings.Join(lines, "\n")
}

// FooterHelpText builds a two-line footer help string:
// line 1 for basic movement, line 2 for shortcut actions.
func FooterHelpText(model help.Model, keys KeyMap) string {
	movement := strings.TrimSpace(model.ShortHelpView([]key.Binding{
		keys.Up,
		keys.Down,
		keys.Left,
		keys.Right,
	}))
	shortcuts := strings.TrimSpace(model.View(&keys))

	switch {
	case movement == "":
		return shortcuts
	case shortcuts == "":
		return movement
	default:
		return movement + "\n" + shortcuts
	}
}
