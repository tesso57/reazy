// Package state holds UI state types for the TUI.
package state

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/tesso57/reazy/internal/domain/reading"
)

// ModelState holds the presentation state for the TUI.
type ModelState struct {
	Session                Session
	FeedList               list.Model
	ArticleList            list.Model
	TextInput              textinput.Model
	Viewport               viewport.Model
	Help                   help.Model
	Spinner                spinner.Model
	Loading                bool
	Keys                   KeyMap
	Width                  int
	Height                 int
	CurrentFeed            *reading.Feed
	Err                    error
	AIStatus               string
	ShowAISummary          bool
	Previous               Session
	DetailParentSession    Session
	History                *reading.History
	Feeds                  []string
	PendingJJExit          bool
	ForceNewsDigestRefresh bool
	NewsTopicDigestGUID    string
	NewsTopicTitle         string
	NewsTopicSummary       string
	NewsTopicTags          []string
}
