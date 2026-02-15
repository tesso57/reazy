// Package intent parses user input into UI intents.
package intent

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
)

// Type represents a user intent.
type Type int

const (
	None Type = iota
	Quit
	ToggleHelp
	AddFeed
	DeleteFeed
	GroupFeeds
	Open
	Back
	Refresh
	Bookmark
	Summarize
	ToggleSummary
)

// Intent represents a parsed user intent.
type Intent struct {
	Type Type
}

// FromKeyMsg maps a key message to an intent.
func FromKeyMsg(msg tea.KeyMsg, keys state.KeyMap) Intent {
	switch {
	case key.Matches(msg, keys.Quit):
		return Intent{Type: Quit}
	case key.Matches(msg, keys.Help):
		return Intent{Type: ToggleHelp}
	case key.Matches(msg, keys.AddFeed):
		return Intent{Type: AddFeed}
	case key.Matches(msg, keys.DeleteFeed):
		return Intent{Type: DeleteFeed}
	case key.Matches(msg, keys.GroupFeeds):
		return Intent{Type: GroupFeeds}
	case key.Matches(msg, keys.Right) || key.Matches(msg, keys.Open):
		return Intent{Type: Open}
	case key.Matches(msg, keys.Left) || key.Matches(msg, keys.Back):
		return Intent{Type: Back}
	case key.Matches(msg, keys.Refresh):
		return Intent{Type: Refresh}
	case key.Matches(msg, keys.Bookmark):
		return Intent{Type: Bookmark}
	case key.Matches(msg, keys.Summarize):
		return Intent{Type: Summarize}
	case key.Matches(msg, keys.ToggleSummary) || msg.String() == "S":
		return Intent{Type: ToggleSummary}
	default:
		return Intent{Type: None}
	}
}
