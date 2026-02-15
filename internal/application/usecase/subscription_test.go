package usecase

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/tesso57/reazy/internal/domain/subscription"
)

type stubSubscriptionRepo struct {
	mock.Mock
	feeds  []string
	groups []subscription.FeedGroup
}

func (s *stubSubscriptionRepo) List() ([]string, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		feeds, _ := args.Get(0).([]string)
		return feeds, args.Error(1)
	}
	out := make([]string, len(s.feeds))
	copy(out, s.feeds)
	return out, nil
}

func (s *stubSubscriptionRepo) Add(url string) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(url)
		return args.Error(0)
	}
	s.feeds = append(s.feeds, url)
	return nil
}

func (s *stubSubscriptionRepo) Remove(index int) error {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called(index)
		return args.Error(0)
	}
	if index < 0 || index >= len(s.feeds) {
		return nil
	}
	s.feeds = append(s.feeds[:index], s.feeds[index+1:]...)
	return nil
}

func (s *stubSubscriptionRepo) ListGroups() ([]subscription.FeedGroup, error) {
	if len(s.ExpectedCalls) > 0 {
		args := s.Called()
		groups, _ := args.Get(0).([]subscription.FeedGroup)
		return groups, args.Error(1)
	}
	if len(s.groups) == 0 {
		return nil, nil
	}
	out := make([]subscription.FeedGroup, 0, len(s.groups))
	for _, group := range s.groups {
		out = append(out, subscription.FeedGroup{
			Name:  group.Name,
			Feeds: append([]string(nil), group.Feeds...),
		})
	}
	return out, nil
}

func (s *stubSubscriptionRepo) ReplaceFeedGroups(groups []subscription.FeedGroup, ungrouped []string) error {
	s.groups = make([]subscription.FeedGroup, 0, len(groups))
	for _, group := range groups {
		s.groups = append(s.groups, subscription.FeedGroup{
			Name:  group.Name,
			Feeds: append([]string(nil), group.Feeds...),
		})
	}
	s.feeds = append([]string(nil), ungrouped...)
	return nil
}

func TestSubscriptionAddTrimsWhitespace(t *testing.T) {
	repo := &stubSubscriptionRepo{}
	svc := NewSubscriptionService(repo)

	_, err := svc.Add("  https://github.com/golang/go/releases.atom\t")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if len(repo.feeds) != 1 || repo.feeds[0] != "https://github.com/golang/go/releases.atom" {
		t.Fatalf("Expected trimmed url in repo feeds, got %#v", repo.feeds)
	}
}

func TestSubscriptionAddRejectsEmpty(t *testing.T) {
	repo := &stubSubscriptionRepo{}
	svc := NewSubscriptionService(repo)

	if _, err := svc.Add(" \t\n"); err == nil {
		t.Fatal("Expected error for empty url")
	}
}

func TestSubscriptionAddRejectsWhitespaceInside(t *testing.T) {
	repo := &stubSubscriptionRepo{}
	svc := NewSubscriptionService(repo)

	if _, err := svc.Add("https://example.com/rss another"); err == nil {
		t.Fatal("Expected error for whitespace in url")
	}
}

func TestSubscriptionListGroups(t *testing.T) {
	repo := &stubSubscriptionRepo{
		groups: []subscription.FeedGroup{
			{Name: "Tech", Feeds: []string{"https://example.com/rss"}},
		},
	}
	svc := NewSubscriptionService(repo)

	groups, supported, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}
	if !supported {
		t.Fatal("expected grouped repository support")
	}
	if len(groups) != 1 || groups[0].Name != "Tech" {
		t.Fatalf("unexpected groups: %#v", groups)
	}
}

func TestSubscriptionReplaceFeedGroups(t *testing.T) {
	repo := &stubSubscriptionRepo{
		feeds: []string{"https://example.com/ungrouped.xml"},
	}
	svc := NewSubscriptionService(repo)

	feeds, supported, err := svc.ReplaceFeedGroups([]subscription.FeedGroup{
		{Name: "Tech", Feeds: []string{"https://example.com/tech.xml"}},
	}, []string{"https://example.com/ungrouped.xml"})
	if err != nil {
		t.Fatalf("ReplaceFeedGroups failed: %v", err)
	}
	if !supported {
		t.Fatal("expected grouped writer support")
	}
	if len(feeds) != 1 || feeds[0] != "https://example.com/ungrouped.xml" {
		t.Fatalf("unexpected feeds: %#v", feeds)
	}
	if len(repo.groups) != 1 || repo.groups[0].Name != "Tech" {
		t.Fatalf("unexpected stored groups: %#v", repo.groups)
	}
}
