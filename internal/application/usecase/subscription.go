// Package usecase contains application-level services.
package usecase

import (
	"fmt"
	"strings"

	"github.com/tesso57/reazy/internal/domain/subscription"
)

// SubscriptionRepository abstracts persistence for feed subscriptions.
type SubscriptionRepository interface {
	List() ([]string, error)
	Add(url string) error
	Remove(index int) error
}

// SubscriptionService provides subscription-related operations.
type SubscriptionService struct {
	Repo SubscriptionRepository
}

type groupedSubscriptionRepository interface {
	ListGroups() ([]subscription.FeedGroup, error)
}

type groupedSubscriptionWriter interface {
	ReplaceFeedGroups(groups []subscription.FeedGroup, ungrouped []string) error
}

// NewSubscriptionService constructs a SubscriptionService.
func NewSubscriptionService(repo SubscriptionRepository) *SubscriptionService {
	return new(SubscriptionService{Repo: repo})
}

// List returns all subscribed feed URLs.
func (s *SubscriptionService) List() ([]string, error) {
	return s.Repo.List()
}

// ListGroups returns configured feed groups when the repository supports it.
func (s *SubscriptionService) ListGroups() ([]subscription.FeedGroup, bool, error) {
	repo, ok := s.Repo.(groupedSubscriptionRepository)
	if !ok {
		return nil, false, nil
	}
	groups, err := repo.ListGroups()
	return groups, true, err
}

// ReplaceFeedGroups persists feed groups and ungrouped feeds when supported.
func (s *SubscriptionService) ReplaceFeedGroups(groups []subscription.FeedGroup, ungrouped []string) ([]string, bool, error) {
	repo, ok := s.Repo.(groupedSubscriptionWriter)
	if !ok {
		return nil, false, nil
	}
	if err := repo.ReplaceFeedGroups(groups, ungrouped); err != nil {
		return nil, true, err
	}
	feeds, err := s.Repo.List()
	return feeds, true, err
}

// Add registers a new feed URL and returns the updated list.
func (s *SubscriptionService) Add(url string) ([]string, error) {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return nil, fmt.Errorf("feed url is empty")
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return nil, fmt.Errorf("feed url contains whitespace")
	}
	if err := s.Repo.Add(trimmed); err != nil {
		return nil, err
	}
	return s.Repo.List()
}

// Remove deletes a feed by index and returns the updated list.
func (s *SubscriptionService) Remove(index int) ([]string, error) {
	if err := s.Repo.Remove(index); err != nil {
		return nil, err
	}
	return s.Repo.List()
}
