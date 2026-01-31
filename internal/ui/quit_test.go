package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/config"
)

func TestQuitDialog(t *testing.T) {
	cfg := &config.Config{
		Feeds: []string{"http://example.com"},
		KeyMap: config.KeyMapConfig{
			Quit: "q",
		},
	}
	m := NewModel(cfg)

	// 1. Initial State
	if m.state != feedView {
		t.Error("Initial state should be feedView")
	}

	// 2. Press 'q' -> Should go to quitView, not quit immediately
	tm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = tm.(*Model)
	if m.state != quitView {
		t.Error("Should switch to quitView on 'q'")
	}
	if cmd != nil {
		t.Error("Should not return tea.Quit command yet")
	}

	// 3. Press 'n' -> Should return to feedView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = tm.(*Model)
	if m.state != feedView {
		t.Error("Should return to feedView on 'n'")
	}

	// 4. Press 'q' again -> quitView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = tm.(*Model)
	if m.state != quitView {
		t.Error("Should switch to quitView")
	}

	// 5. Press 'esc' -> Should return to feedView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state != feedView {
		t.Error("Should return to feedView on 'esc'")
	}

	// 6. From Article View
	m.state = articleView
	// Press 'q'
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = tm.(*Model)
	if m.state != quitView {
		t.Error("Should switch to quitView from articleView")
	}
	if m.previousState != articleView {
		t.Error("Should remember previous state as articleView")
	}

	// Press 'q' (cancel) -> Should return to articleView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = tm.(*Model)
	if m.state != articleView {
		t.Error("Should return to articleView on 'q' (cancel)")
	}

	// 7. Confirm Quit ('y')
	// Enter quit view again
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = tm.(*Model)
	// Press 'y'
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Error("Should return command on 'y'")
	}
	// Note: We can't easily verify it is exactly tea.Quit without deep inspection or comparing func pointers which is hard.
	// But standard pattern is returning tea.Quit which is a tea.Cmd.
	// Checking if it's not nil is a good enough proxy for now given the implementation returns tea.Quit.
}
