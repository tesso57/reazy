package usecase

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

type stubSubscriptionRepo struct {
	mock.Mock
	feeds []string
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
