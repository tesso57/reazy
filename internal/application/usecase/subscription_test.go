package usecase

import "testing"

type stubSubscriptionRepo struct {
	feeds []string
	added string
	err   error
}

func (s *stubSubscriptionRepo) List() ([]string, error) {
	out := make([]string, len(s.feeds))
	copy(out, s.feeds)
	return out, s.err
}

func (s *stubSubscriptionRepo) Add(url string) error {
	s.added = url
	s.feeds = append(s.feeds, url)
	return s.err
}

func (s *stubSubscriptionRepo) Remove(index int) error {
	if index < 0 || index >= len(s.feeds) {
		return s.err
	}
	s.feeds = append(s.feeds[:index], s.feeds[index+1:]...)
	return s.err
}

func TestSubscriptionAddTrimsWhitespace(t *testing.T) {
	repo := &stubSubscriptionRepo{}
	svc := SubscriptionService{Repo: repo}

	_, err := svc.Add("  https://github.com/golang/go/releases.atom\t")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if repo.added != "https://github.com/golang/go/releases.atom" {
		t.Fatalf("Expected trimmed url, got %q", repo.added)
	}
}

func TestSubscriptionAddRejectsEmpty(t *testing.T) {
	repo := &stubSubscriptionRepo{}
	svc := SubscriptionService{Repo: repo}

	if _, err := svc.Add(" \t\n"); err == nil {
		t.Fatal("Expected error for empty url")
	}
}

func TestSubscriptionAddRejectsWhitespaceInside(t *testing.T) {
	repo := &stubSubscriptionRepo{}
	svc := SubscriptionService{Repo: repo}

	if _, err := svc.Add("https://example.com/rss another"); err == nil {
		t.Fatal("Expected error for whitespace in url")
	}
}
