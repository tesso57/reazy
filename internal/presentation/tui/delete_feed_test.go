package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/presentation/tui/presenter"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
)

func TestDeleteFeedDialog(t *testing.T) {
	cfg := settings.Settings{
		Feeds: []string{"http://example.com/1", "http://example.com/2"},
		KeyMap: settings.KeyMapConfig{
			DeleteFeed: "x",
		},
	}
	m := newTestModel(cfg, &stubSubscriptionRepo{feeds: cfg.Feeds}, &stubHistoryRepo{}, stubFeedFetcher{})

	// 1. Initial State
	if m.state.Session != state.FeedView {
		t.Error("Initial state should be feedView")
	}

	// Move to first feed (index 1, as 0 is "All Feeds")
	m.state.FeedList.Select(1)
	if _, ok := m.state.FeedList.SelectedItem().(*presenter.Item); !ok {
		t.Fatal("Should have selected an item")
	}

	// 2. Press 'x' -> Should go to deleteFeedView
	tm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = tm.(*Model)
	if m.state.Session != state.DeleteFeedView {
		t.Error("Should switch to DeleteFeedView on 'x'")
	}

	// 3. Press 'n' -> Should return to feedView without deleting
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = tm.(*Model)
	if m.state.Session != state.FeedView {
		t.Error("Should return to feedView on 'n'")
	}
	if len(m.state.Feeds) != 2 {
		t.Errorf("Should keep 2 feeds, got %d", len(m.state.Feeds))
	}

	// 4. Press 'x' again -> deleteFeedView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = tm.(*Model)
	if m.state.Session != state.DeleteFeedView {
		t.Error("Should switch to DeleteFeedView")
	}

	// 5. Press 'esc' -> Should return to feedView
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = tm.(*Model)
	if m.state.Session != state.FeedView {
		t.Error("Should return to feedView on 'esc'")
	}

	// 6. Confirm Delete ('y')
	// Select item again just in case
	m.state.FeedList.Select(1)
	// Enter delete view
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = tm.(*Model)
	// Press 'y'
	tm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = tm.(*Model)

	if m.state.Session != state.FeedView {
		t.Error("Should return to feedView after deletion")
	}
	if len(m.state.Feeds) != 1 {
		t.Errorf("Should have 1 feed remaining, got %d", len(m.state.Feeds))
	}
	if m.state.Feeds[0] != "http://example.com/2" {
		t.Errorf("Should have deleted the first feed, remaining: %s", m.state.Feeds[0])
	}
}
