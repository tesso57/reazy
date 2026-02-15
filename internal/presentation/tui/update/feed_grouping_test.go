package update

import (
	"context"
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/tesso57/reazy/internal/application/usecase"
	"github.com/tesso57/reazy/internal/domain/subscription"
	"github.com/tesso57/reazy/internal/presentation/tui/state"
)

type updateFeedGroupingGenerator struct {
	groups []subscription.FeedGroup
	err    error
}

func (g updateFeedGroupingGenerator) Generate(_ context.Context, _ usecase.FeedGroupingRequest) ([]subscription.FeedGroup, error) {
	return g.groups, g.err
}

type updateSubscriptionRepo struct {
	feeds  []string
	groups []subscription.FeedGroup
}

func (r *updateSubscriptionRepo) List() ([]string, error) {
	out := make([]string, 0, len(r.feeds))
	for _, group := range r.groups {
		out = append(out, group.Feeds...)
	}
	out = append(out, r.feeds...)
	return out, nil
}

func (r *updateSubscriptionRepo) Add(url string) error {
	r.feeds = append(r.feeds, url)
	return nil
}

func (r *updateSubscriptionRepo) Remove(index int) error {
	feeds, _ := r.List()
	if index < 0 || index >= len(feeds) {
		return nil
	}
	remaining := index
	for groupIndex := range r.groups {
		groupLen := len(r.groups[groupIndex].Feeds)
		if remaining < groupLen {
			feeds := r.groups[groupIndex].Feeds
			r.groups[groupIndex].Feeds = append(feeds[:remaining], feeds[remaining+1:]...)
			return nil
		}
		remaining -= groupLen
	}
	r.feeds = append(r.feeds[:remaining], r.feeds[remaining+1:]...)
	return nil
}

func (r *updateSubscriptionRepo) ListGroups() ([]subscription.FeedGroup, error) {
	if len(r.groups) == 0 {
		return nil, nil
	}
	out := make([]subscription.FeedGroup, 0, len(r.groups))
	for _, group := range r.groups {
		out = append(out, subscription.FeedGroup{
			Name:  group.Name,
			Feeds: append([]string(nil), group.Feeds...),
		})
	}
	return out, nil
}

func (r *updateSubscriptionRepo) ReplaceFeedGroups(groups []subscription.FeedGroup, ungrouped []string) error {
	r.groups = make([]subscription.FeedGroup, 0, len(groups))
	for _, group := range groups {
		r.groups = append(r.groups, subscription.FeedGroup{
			Name:  group.Name,
			Feeds: append([]string(nil), group.Feeds...),
		})
	}
	r.feeds = append([]string(nil), ungrouped...)
	return nil
}

func TestGenerateFeedGroupingCmd(t *testing.T) {
	repo := &updateSubscriptionRepo{
		feeds: []string{"https://planetpython.org/rss20.xml"},
	}
	subscriptions := usecase.NewSubscriptionService(repo)
	grouping := usecase.NewFeedGroupingService(updateFeedGroupingGenerator{
		groups: []subscription.FeedGroup{
			{
				Name:  "Tech",
				Feeds: []string{"https://news.ycombinator.com/rss"},
			},
		},
	})

	cmd := GenerateFeedGroupingCmd(grouping, subscriptions, []string{
		"https://news.ycombinator.com/rss",
		"https://planetpython.org/rss20.xml",
	})
	raw := cmd()
	msg, ok := raw.(FeedGroupingCompletedMsg)
	if !ok {
		t.Fatalf("unexpected cmd message type: %#v", raw)
	}
	if msg.Err != nil {
		t.Fatalf("FeedGroupingCompletedMsg.Err = %v", msg.Err)
	}
	if len(msg.Groups) != 1 || msg.Groups[0].Name != "Tech" {
		t.Fatalf("unexpected groups: %#v", msg.Groups)
	}
	if len(msg.Ungrouped) != 1 || msg.Ungrouped[0] != "https://planetpython.org/rss20.xml" {
		t.Fatalf("unexpected ungrouped: %#v", msg.Ungrouped)
	}
}

func TestGenerateFeedGroupingCmd_Disabled(t *testing.T) {
	cmd := GenerateFeedGroupingCmd(nil, nil, []string{"https://a.example.com/rss", "https://b.example.com/rss"})
	raw := cmd()
	msg, ok := raw.(FeedGroupingCompletedMsg)
	if !ok {
		t.Fatalf("unexpected cmd message type: %#v", raw)
	}
	if msg.Err == nil {
		t.Fatal("expected disabled error")
	}
}

func TestHandleFeedGroupingCompletedMsg(t *testing.T) {
	s := &state.ModelState{
		FeedList: list.New(nil, list.NewDefaultDelegate(), 80, 20),
	}

	HandleFeedGroupingCompletedMsg(s, FeedGroupingCompletedMsg{
		Feeds: []string{"https://news.ycombinator.com/rss", "https://planetpython.org/rss20.xml"},
		Groups: []subscription.FeedGroup{
			{Name: "Tech", Feeds: []string{"https://news.ycombinator.com/rss"}},
		},
		Ungrouped: []string{"https://planetpython.org/rss20.xml"},
	})

	if s.Err != nil {
		t.Fatalf("state err should be nil, got %v", s.Err)
	}
	if len(s.FeedGroups) != 1 || s.FeedGroups[0].Name != "Tech" {
		t.Fatalf("unexpected feed groups: %#v", s.FeedGroups)
	}
	if s.StatusMessage == "" {
		t.Fatal("status message should not be empty")
	}
}

func TestHandleFeedGroupingCompletedMsg_Error(t *testing.T) {
	s := &state.ModelState{
		FeedList: list.New(nil, list.NewDefaultDelegate(), 80, 20),
	}

	HandleFeedGroupingCompletedMsg(s, FeedGroupingCompletedMsg{
		Err: errors.New("failed"),
	})

	if s.Err == nil {
		t.Fatal("expected error to be set")
	}
	if s.StatusMessage == "" {
		t.Fatal("expected failure status message")
	}
}
