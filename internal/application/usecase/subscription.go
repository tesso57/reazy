// Package usecase contains application-level services.
package usecase

import (
	"fmt"
	"strings"
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

// NewSubscriptionService constructs a SubscriptionService.
func NewSubscriptionService(repo SubscriptionRepository) SubscriptionService {
	return SubscriptionService{Repo: repo}
}

// List returns all subscribed feed URLs.
func (s SubscriptionService) List() ([]string, error) {
	return s.Repo.List()
}

// Add registers a new feed URL and returns the updated list.
func (s SubscriptionService) Add(url string) ([]string, error) {
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
func (s SubscriptionService) Remove(index int) ([]string, error) {
	if err := s.Repo.Remove(index); err != nil {
		return nil, err
	}
	return s.Repo.List()
}
