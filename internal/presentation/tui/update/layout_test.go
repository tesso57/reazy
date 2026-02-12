package update

import (
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/tesso57/reazy/internal/application/settings"
	"github.com/tesso57/reazy/internal/presentation/tui/metrics"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
)

func TestFooterHeight_ReflectsAIStatusInFooter(t *testing.T) {
	s := newLayoutTestState()
	s.Width = 100
	s.Session = state.ArticleView

	base := footerHeight(s)
	s.AIStatus = "AI: updated 2026-02-09 10:00"
	withStatus := footerHeight(s)
	if withStatus <= base {
		t.Fatalf("footer height should grow with AI status in article view: base=%d with=%d", base, withStatus)
	}

	s.Session = state.FeedView
	inFeedView := footerHeight(s)
	if inFeedView != base {
		t.Fatalf("footer height should ignore AI status in feed view: base=%d feed=%d", base, inFeedView)
	}
}

func TestBuildLayoutMetrics_MainWidthSubtractsSidebarBorder(t *testing.T) {
	s := newLayoutTestState()
	s.Width = 120
	s.Height = 40

	layout := buildLayoutMetrics(s)
	sidebarWidth := s.Width / 3
	wantMainWidth := clampMin(s.Width-sidebarWidth-metrics.SidebarRightBorderWidth, 1)
	if layout.mainWidth != wantMainWidth {
		t.Fatalf("main width = %d, want %d", layout.mainWidth, wantMainWidth)
	}
}

func newLayoutTestState() *state.ModelState {
	keys := state.NewKeyMap(settings.KeyMapConfig{
		Up: "k", Down: "j", Left: "h", Right: "l",
		Open: "enter", Back: "esc", Quit: "q",
		AddFeed: "a", DeleteFeed: "x", Refresh: "r", Bookmark: "b",
		Summarize: "s", ToggleSummary: "S",
		UpPage: "pgup", DownPage: "pgdn", Top: "g", Bottom: "G",
	})
	return &state.ModelState{
		Session:     state.FeedView,
		Help:        help.New(),
		Keys:        keys,
		FeedList:    list.New(nil, list.NewDefaultDelegate(), 0, 0),
		ArticleList: list.New(nil, list.NewDefaultDelegate(), 0, 0),
		Width:       100,
		Height:      40,
	}
}
