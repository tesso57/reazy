// Package usecase contains application-level services.
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/tesso57/reazy/internal/domain/subscription"
)

const maxFeedGroupingFeeds = 200

// FeedGroupingFeed is one input feed for AI grouping.
type FeedGroupingFeed struct {
	URL  string `json:"url"`
	Host string `json:"host"`
	Path string `json:"path"`
}

// FeedGroupingRequest is the payload for AI feed grouping.
type FeedGroupingRequest struct {
	Feeds []FeedGroupingFeed `json:"feeds"`
}

// FeedGroupingResult contains grouped and ungrouped feeds.
type FeedGroupingResult struct {
	Groups    []subscription.FeedGroup
	Ungrouped []string
}

// FeedGroupingGenerator abstracts AI grouping generation.
type FeedGroupingGenerator interface {
	Generate(ctx context.Context, req FeedGroupingRequest) ([]subscription.FeedGroup, error)
}

// FeedGroupingService coordinates AI grouping.
type FeedGroupingService struct {
	Generator FeedGroupingGenerator
}

// NewFeedGroupingService constructs a FeedGroupingService.
func NewFeedGroupingService(generator FeedGroupingGenerator) *FeedGroupingService {
	return new(FeedGroupingService{Generator: generator})
}

// Enabled reports whether AI grouping is available.
func (s *FeedGroupingService) Enabled() bool {
	return s != nil && s.Generator != nil
}

// Group builds feed groups from feed URLs.
func (s *FeedGroupingService) Group(ctx context.Context, feeds []string) (FeedGroupingResult, error) {
	if !s.Enabled() {
		return FeedGroupingResult{}, errors.New("codex integration is disabled")
	}

	normalizedFeeds := normalizeFeedURLList(feeds)
	if len(normalizedFeeds) < 2 {
		return FeedGroupingResult{}, errors.New("at least 2 feeds are required for grouping")
	}

	req := buildFeedGroupingRequest(normalizedFeeds)
	suggestedGroups, err := s.Generator.Generate(ctx, req)
	if err != nil {
		return FeedGroupingResult{}, err
	}

	groups := normalizeSuggestedFeedGroups(suggestedGroups, normalizedFeeds)
	if len(groups) == 0 {
		return FeedGroupingResult{}, errors.New("feed grouping returned no valid groups")
	}

	assigned := make(map[string]struct{}, len(normalizedFeeds))
	for _, group := range groups {
		for _, feedURL := range group.Feeds {
			assigned[feedURL] = struct{}{}
		}
	}

	ungrouped := make([]string, 0, len(normalizedFeeds))
	for _, feedURL := range normalizedFeeds {
		if _, ok := assigned[feedURL]; ok {
			continue
		}
		ungrouped = append(ungrouped, feedURL)
	}

	return FeedGroupingResult{
		Groups:    groups,
		Ungrouped: ungrouped,
	}, nil
}

func normalizeFeedURLList(feeds []string) []string {
	if len(feeds) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(feeds))
	normalized := make([]string, 0, min(len(feeds), maxFeedGroupingFeeds))
	for _, feedURL := range feeds {
		trimmed := strings.TrimSpace(feedURL)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
		if len(normalized) >= maxFeedGroupingFeeds {
			break
		}
	}
	return normalized
}

func buildFeedGroupingRequest(feeds []string) FeedGroupingRequest {
	result := FeedGroupingRequest{
		Feeds: make([]FeedGroupingFeed, 0, len(feeds)),
	}
	for _, feedURL := range feeds {
		host := ""
		path := ""
		if parsed, err := url.Parse(feedURL); err == nil {
			host = strings.TrimSpace(parsed.Hostname())
			path = strings.Trim(strings.TrimSpace(parsed.Path), "/")
		}
		result.Feeds = append(result.Feeds, FeedGroupingFeed{
			URL:  feedURL,
			Host: host,
			Path: path,
		})
	}
	return result
}

func normalizeSuggestedFeedGroups(groups []subscription.FeedGroup, feeds []string) []subscription.FeedGroup {
	if len(groups) == 0 || len(feeds) == 0 {
		return nil
	}

	validFeeds := make(map[string]struct{}, len(feeds))
	for _, feedURL := range feeds {
		validFeeds[feedURL] = struct{}{}
	}

	assigned := make(map[string]struct{}, len(feeds))
	result := make([]subscription.FeedGroup, 0, len(groups))
	groupIndexByName := map[string]int{}

	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		if name == "" {
			continue
		}

		selectedFeeds := make([]string, 0, len(group.Feeds))
		seenInGroup := make(map[string]struct{}, len(group.Feeds))
		for _, feedURL := range group.Feeds {
			feedURL = strings.TrimSpace(feedURL)
			if feedURL == "" {
				continue
			}
			if _, ok := validFeeds[feedURL]; !ok {
				continue
			}
			if _, alreadyAssigned := assigned[feedURL]; alreadyAssigned {
				continue
			}
			if _, dup := seenInGroup[feedURL]; dup {
				continue
			}
			seenInGroup[feedURL] = struct{}{}
			selectedFeeds = append(selectedFeeds, feedURL)
		}
		if len(selectedFeeds) == 0 {
			continue
		}

		key := strings.ToLower(name)
		if idx, ok := groupIndexByName[key]; ok {
			result[idx].Feeds = append(result[idx].Feeds, selectedFeeds...)
		} else {
			result = append(result, subscription.FeedGroup{
				Name:  name,
				Feeds: selectedFeeds,
			})
			groupIndexByName[key] = len(result) - 1
		}

		for _, feedURL := range selectedFeeds {
			assigned[feedURL] = struct{}{}
		}
	}

	return result
}

// PromptFeedGroupingGenerator builds prompts and parses feed grouping JSON output.
type PromptFeedGroupingGenerator struct {
	Client TextGenerator
}

// NewPromptFeedGroupingGenerator constructs a PromptFeedGroupingGenerator.
func NewPromptFeedGroupingGenerator(client TextGenerator) PromptFeedGroupingGenerator {
	return PromptFeedGroupingGenerator{Client: client}
}

// Generate implements FeedGroupingGenerator.
func (g PromptFeedGroupingGenerator) Generate(ctx context.Context, req FeedGroupingRequest) ([]subscription.FeedGroup, error) {
	if g.Client == nil {
		return nil, errors.New("ai client is not configured")
	}
	raw, err := g.Client.Generate(ctx, buildFeedGroupingPrompt(req))
	if err != nil {
		return nil, err
	}
	return parseFeedGroupingOutput(raw)
}

func buildFeedGroupingPrompt(req FeedGroupingRequest) string {
	payload := FeedGroupingRequest{
		Feeds: make([]FeedGroupingFeed, 0, len(req.Feeds)),
	}
	for _, feed := range req.Feeds {
		payload.Feeds = append(payload.Feeds, FeedGroupingFeed{
			URL:  strings.TrimSpace(feed.URL),
			Host: strings.TrimSpace(feed.Host),
			Path: strings.TrimSpace(feed.Path),
		})
	}
	data, _ := json.Marshal(payload)

	return strings.Join([]string{
		"You are helping an RSS reader organize feed subscriptions.",
		"Propose concise feed groups based on feed URL/host/path hints.",
		`Return ONLY valid JSON without markdown: {"groups":[{"name":"...","feeds":["..."]}]}`,
		"Rules:",
		"- group name: concise and clear (Japanese or English).",
		"- feeds: include ONLY URLs provided in input.",
		"- each feed must appear in at most one group.",
		"- omit uncertain feeds instead of forcing a wrong group.",
		"- return the best possible grouping even if partial.",
		"Input JSON:",
		string(data),
	}, "\n")
}

func parseFeedGroupingOutput(raw string) ([]subscription.FeedGroup, error) {
	type groupPayload struct {
		Name  string   `json:"name"`
		Feeds []string `json:"feeds"`
	}
	type payload struct {
		Groups []groupPayload `json:"groups"`
	}

	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, errors.New("ai client returned empty output")
	}

	tryDecode := func(data string) ([]subscription.FeedGroup, error) {
		var out payload
		if err := json.Unmarshal([]byte(data), &out); err != nil {
			return nil, err
		}
		result := make([]subscription.FeedGroup, 0, len(out.Groups))
		for _, group := range out.Groups {
			result = append(result, subscription.FeedGroup{
				Name:  group.Name,
				Feeds: group.Feeds,
			})
		}
		return result, nil
	}

	groups, err := tryDecode(text)
	if err == nil {
		return groups, nil
	}

	jsonObject := extractInsightJSONObject(text)
	if jsonObject == "" {
		return nil, fmt.Errorf("failed to parse ai output as JSON: %w", err)
	}
	groups, decodeErr := tryDecode(jsonObject)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to parse ai output as JSON: %w", decodeErr)
	}
	return groups, nil
}
