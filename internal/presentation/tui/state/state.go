// Package state holds UI state types for the TUI.
package state

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/tesso57/reazy/internal/application/settings"
)

// Session represents the current view state.
type Session int

const (
	FeedView Session = iota
	ArticleView
	NewsTopicView
	DetailView
	AddingFeedView
	DeleteFeedView
	QuitView
)

// KeyMap defines the keybindings for the application.
type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	UpPage        key.Binding
	DownPage      key.Binding
	Top           key.Binding
	Bottom        key.Binding
	Open          key.Binding
	Back          key.Binding
	Quit          key.Binding
	AddFeed       key.Binding
	DeleteFeed    key.Binding
	Refresh       key.Binding
	Bookmark      key.Binding
	Summarize     key.Binding
	ToggleSummary key.Binding
	Help          key.Binding
}

// ShortHelp returns a subset of keybindings for the help view.
func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit, k.Back, k.Open}
}

// FullHelp returns all keybindings for the help view.
func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Top, k.Bottom, k.UpPage, k.DownPage},
		{k.Open, k.Back, k.Quit},
		{k.AddFeed, k.DeleteFeed, k.Refresh, k.Bookmark},
		{k.Summarize, k.ToggleSummary, k.Help},
	}
}

// NewKeyMap creates a new KeyMap from the configuration.
func NewKeyMap(cfg settings.KeyMapConfig) KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Up)...),
			key.WithHelp(cfg.Up, "up"),
		),
		Down: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Down)...),
			key.WithHelp(cfg.Down, "down"),
		),
		Left: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Left)...),
			key.WithHelp(cfg.Left, "back/feeds"),
		),
		Right: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Right)...),
			key.WithHelp(cfg.Right, "details"),
		),
		UpPage: key.NewBinding(
			key.WithKeys(splitKeys(cfg.UpPage)...),
			key.WithHelp(cfg.UpPage, "pgup"),
		),
		DownPage: key.NewBinding(
			key.WithKeys(splitKeys(cfg.DownPage)...),
			key.WithHelp(cfg.DownPage, "pgdn"),
		),
		Top: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Top)...),
			key.WithHelp(cfg.Top, "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Bottom)...),
			key.WithHelp(cfg.Bottom, "bottom"),
		),
		Open: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Open)...),
			key.WithHelp(cfg.Open, "open"),
		),
		Back: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Back)...),
			key.WithHelp(cfg.Back, "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Quit)...),
			key.WithHelp(cfg.Quit, "quit"),
		),
		AddFeed: key.NewBinding(
			key.WithKeys(splitKeys(cfg.AddFeed)...),
			key.WithHelp(cfg.AddFeed, "add"),
		),
		DeleteFeed: key.NewBinding(
			key.WithKeys(splitKeys(cfg.DeleteFeed)...),
			key.WithHelp(cfg.DeleteFeed, "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Refresh)...),
			key.WithHelp(cfg.Refresh, "refresh"),
		),
		Bookmark: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Bookmark)...),
			key.WithHelp(cfg.Bookmark, "bookmark"),
		),
		Summarize: key.NewBinding(
			key.WithKeys(splitKeys(cfg.Summarize)...),
			key.WithHelp(cfg.Summarize, "ai summary"),
		),
		ToggleSummary: key.NewBinding(
			key.WithKeys(splitKeys(cfg.ToggleSummary)...),
			key.WithHelp(cfg.ToggleSummary, "toggle summary"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

func splitKeys(keys string) []string {
	parts := strings.Split(keys, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		keyName := strings.TrimSpace(part)
		if keyName == "" {
			continue
		}
		out = append(out, keyName)
		switch keyName {
		case "pgdn":
			out = append(out, "pgdown")
		case "pgdown":
			out = append(out, "pgdn")
		}
	}
	return out
}
